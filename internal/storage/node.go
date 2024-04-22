package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	NodeAttrKind = "sys.kind"
)

const (
	NodeAttrKindRoot = "root"
)

type Node struct {
	ID              string
	Name            string
	ContentHash     string
	ContentMimetype string
	ContentLength   int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       time.Time

	Attributes []NodeAttribute
}

func (n *Node) IsDeleted() bool {
	return !n.DeletedAt.IsZero()
}

type nodeRow struct {
	ID              string    `db:"id"`
	Name            string    `db:"name"`
	ContentHash     string    `db:"content_hash"`
	ContentMimetype string    `db:"content_mimetype"`
	ContentLength   int64     `db:"content_length"`
	CreatedAt       Timestamp `db:"created_at"`
	UpdatedAt       Timestamp `db:"updated_at"`
	DeletedAt       Timestamp `db:"deleted_at"`
}

type NodeAttribute struct {
	Key       string
	Value     string
	CreatedAt time.Time
}

type nodeAttributeRow struct {
	NodeID    string    `db:"node_id"`
	Key       string    `db:"key"`
	Value     string    `db:"value"`
	CreatedAt Timestamp `db:"created_at"`
}

func (s *Storage) GenerateNodeID(ctx context.Context) (string, error) {
	// TODO: configurable ID generation.
	nodeID := time.Now().Format("20060102-150405")
	s.nodeIDMu.Lock()
	defer s.nodeIDMu.Unlock()
	if nodeID == s.nodeIDLast {
		s.nodeIDCounter += 1
		nodeID = fmt.Sprintf("%s-%d", nodeID, s.nodeIDCounter)
	} else {
		s.nodeIDLast = nodeID
		s.nodeIDCounter = 0
	}
	return nodeID, nil
}

func (s *Storage) NodeSave(ctx context.Context, node Node) error {
	now := time.Now()
	row := nodeRow{
		ID:              node.ID,
		Name:            node.Name,
		ContentHash:     node.ContentHash,
		ContentMimetype: node.ContentMimetype,
		CreatedAt:       Timestamp{now},
		UpdatedAt:       Timestamp{now},
	}
	attrRows := make([]nodeAttributeRow, 0, len(node.Attributes))
	for _, attr := range node.Attributes {
		attrRows = append(attrRows, nodeAttributeRow{
			NodeID:    node.ID,
			Key:       attr.Key,
			Value:     attr.Value,
			CreatedAt: Timestamp{now},
		})
	}

	txCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	tx, err := s.writeDB.BeginTxx(txCtx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	_, err = tx.NamedExecContext(
		txCtx,
		`INSERT INTO node(id, name, content_hash, content_mimetype, created_at, updated_at)
			VALUES (:id, :name, :content_hash, :content_mimetype, :created_at, :updated_at)
			ON CONFLICT(id) DO UPDATE
				SET name=excluded.name,
					content_hash=excluded.content_hash,
					content_mimetype=excluded.content_mimetype,
					updated_at=excluded.updated_at`,
		&row,
	)
	if err != nil {
		return fmt.Errorf("upsert node: %w", err)
	}

	// Replace node attributes. Already existed attributes will preserve created_at value.
	preservedAttrs := make([]string, 0, len(attrRows))
	for _, r := range attrRows {
		preservedAttrs = append(preservedAttrs, r.Key+"="+r.Value)
	}
	var query string
	var args []any
	if len(preservedAttrs) > 0 {
		query, args, err = sqlx.In(
			`DELETE FROM node_attribute WHERE node_id = ? AND key || '=' || value NOT IN (?)`,
			node.ID,
			preservedAttrs,
		)
	} else {
		query, args, err = sqlx.In(
			`DELETE FROM node_attribute WHERE node_id = ?`,
			node.ID,
		)
	}
	if err != nil {
		return fmt.Errorf("prepare delete attrs query: %w", err)
	}
	query = tx.Rebind(query)
	_, err = tx.ExecContext(txCtx, query, args...)
	if err != nil {
		return fmt.Errorf("delete attrs: %w", err)
	}
	if len(attrRows) > 0 {
		_, err = tx.NamedExecContext(
			ctx,
			`INSERT INTO node_attribute(node_id, key, value, created_at)
				VALUES (:node_id, :key, :value, :created_at)
				ON CONFLICT DO NOTHING`,
			attrRows,
		)
		if err != nil {
			return fmt.Errorf("insert attrs: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (s *Storage) NodesLoad(ctx context.Context, ids []string) (map[string]Node, error) {
	query, args, err := sqlx.In(
		`SELECT node_id, key, value, created_at FROM node_attribute WHERE node_id IN (?)`, ids,
	)
	if err != nil {
		return nil, fmt.Errorf("prepare attrs query: %w", err)
	}
	query = s.readDB.Rebind(query)
	rows, err := s.readDB.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select node: %w", err)
	}
	attrs := make(map[string][]NodeAttribute)
	attrRow := nodeAttributeRow{}
	for rows.Next() {
		err := rows.StructScan(&attrRow)
		if err != nil {
			return nil, fmt.Errorf("scan attr: %w", errors.Join(err, rows.Close()))
		}
		attrs[attrRow.NodeID] = append(attrs[attrRow.NodeID], NodeAttribute{
			Key:       attrRow.Key,
			Value:     attrRow.Value,
			CreatedAt: attrRow.CreatedAt.Time,
		})
	}

	query, args, err = sqlx.In(
		`SELECT n.id, n.name, n.content_hash, n.content_mimetype, n.created_at, n.updated_at, n.deleted_at,
				octet_length(c.content) AS content_length
			FROM node AS n
			JOIN node_content AS c
			WHERE n.id IN (?) AND c.hash = n.content_hash`,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("prepare node query: %w", err)
	}
	query = s.readDB.Rebind(query)
	rows, err = s.readDB.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select node: %w", err)
	}
	nodes := make(map[string]Node, len(ids))
	row := nodeRow{}
	for rows.Next() {
		err := rows.StructScan(&row)
		if err != nil {
			return nil, fmt.Errorf("scan node: %w", errors.Join(err, rows.Close()))
		}
		nodes[row.ID] = Node{
			ID:              row.ID,
			Name:            row.Name,
			ContentHash:     row.ContentHash,
			ContentMimetype: row.ContentMimetype,
			ContentLength:   row.ContentLength,
			CreatedAt:       row.CreatedAt.Time,
			UpdatedAt:       row.UpdatedAt.Time,
			DeletedAt:       row.DeletedAt.Time,
			Attributes:      attrs[row.ID],
		}
	}

	return nodes, nil
}
