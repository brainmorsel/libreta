package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	EdgeRelChild = "child"
	EdgeRelLink  = "link"
	EdgeRelChain = "chain"
)

type Edge struct {
	SrcID     string
	DstID     string
	Relation  string
	CreatedAt time.Time
}

type edgeRow struct {
	SrcID     string    `db:"src_id"`
	DstID     string    `db:"dst_id"`
	Relation  string    `db:"relation"`
	CreatedAt Timestamp `db:"created_at"`
}

func (s *Storage) EdgesAdd(ctx context.Context, edges []Edge) error {
	now := time.Now()
	rows := make([]edgeRow, 0, len(edges))
	for _, edge := range edges {
		rows = append(rows, edgeRow{
			SrcID:     edge.SrcID,
			DstID:     edge.DstID,
			Relation:  edge.Relation,
			CreatedAt: Timestamp{now},
		})
	}
	_, err := s.writeDB.NamedExecContext(
		ctx,
		`INSERT INTO edge(src_id, dst_id, relation, created_at)
			VALUES (:src_id, :dst_id, :relation, :created_at)
			ON CONFLICT DO NOTHING`,
		rows,
	)
	if err != nil {
		return fmt.Errorf("insert edges: %w", err)
	}

	return nil
}

func (s *Storage) EdgesRemove(ctx context.Context, edges []Edge) error {
	rows := make([]edgeRow, 0, len(edges))
	for _, edge := range edges {
		rows = append(rows, edgeRow{
			SrcID:    edge.SrcID,
			DstID:    edge.DstID,
			Relation: edge.Relation,
		})
	}
	_, err := s.writeDB.NamedExecContext(
		ctx,
		`WITH remove(src_id, dst_id, relation) AS (VALUES (:src_id, :dst_id, :relation))
			DELETE FROM edge WHERE EXISTS (
				SELECT 1 FROM remove WHERE edge.src_id = remove.src_id AND edge.dst_id = remove.dst_id AND edge.relation = remove.relation)`,
		rows,
	)
	if err != nil {
		return fmt.Errorf("delete edges: %w", err)
	}
	return nil
}

func sqlTupleList(tupleLen, tuplesCount int) string {
	tupleSB := strings.Builder{}
	tupleSB.WriteString("(")
	for n := range tupleLen {
		tupleSB.WriteString("?")
		if n < tupleLen-1 {
			tupleSB.WriteString(",")
		}
	}
	tupleSB.WriteString(")")
	tupleS := tupleSB.String()
	valuesSB := strings.Builder{}
	for n := range tuplesCount {
		valuesSB.WriteString(tupleS)
		if n < tuplesCount-1 {
			valuesSB.WriteString(",")
		}
	}
	return valuesSB.String()
}

func (s *Storage) EdgesForNodes(ctx context.Context, nodeIDs []string) ([]Edge, error) {
	query := `WITH ids(id) AS (VALUES ` + sqlTupleList(1, len(nodeIDs)) + `)
			SELECT src_id, dst_id, relation, created_at
				FROM edge
				WHERE src_id IN (SELECT id FROM ids) OR dst_id IN (SELECT id FROM ids)`
	args := make([]any, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		args = append(args, nodeID)
	}
	rows, err := s.readDB.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select edges: %w", err)
	}
	edges := make([]Edge, 0)
	row := edgeRow{}
	for rows.Next() {
		err := rows.StructScan(&row)
		if err != nil {
			return nil, fmt.Errorf("scan edge: %w", errors.Join(err, rows.Close()))
		}
		edges = append(edges, Edge{
			SrcID:     row.SrcID,
			DstID:     row.DstID,
			Relation:  row.Relation,
			CreatedAt: row.CreatedAt.Time,
		})
	}
	return edges, nil
}
