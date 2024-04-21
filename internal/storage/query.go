package storage

import (
	"context"
	"errors"
	"fmt"
)

func (s *Storage) QueryByAttribute(ctx context.Context, key, value string) ([]string, error) {
	rows, err := s.readDB.NamedQueryContext(
		ctx,
		`SELECT a.node_id FROM node_attribute AS a JOIN node AS n
			WHERE a.key = :key AND a.value = :value AND a.node_id = n.id AND n.deleted_at IS NULL`,
		map[string]any{
			"key":   key,
			"value": value,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("select node ids: %w", err)
	}
	nodeIDs := make([]string, 0)
	var nodeID string
	for rows.Next() {
		if err := rows.Scan(&nodeID); err != nil {
			return nil, fmt.Errorf("scan node id: %w", errors.Join(err, rows.Close()))
		}
		nodeIDs = append(nodeIDs, nodeID)
	}
	return nodeIDs, nil
}

func (s *Storage) QueryFullTextSearch(ctx context.Context, searchTerm string, limit int) ([]string, error) {
	// TODO: implement snowball stemming pre-processing.
	rows, err := s.readDB.NamedQueryContext(
		ctx,
		`SELECT id FROM node_fts_idx WHERE node_fts_idx MATCH :search_term ORDER BY rank LIMIT :limit`,
		map[string]any{
			"search_term": searchTerm,
			"limit":       limit,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("select node ids: %w", err)
	}
	nodeIDs := make([]string, 0)
	var nodeID string
	for rows.Next() {
		if err := rows.Scan(&nodeID); err != nil {
			return nil, fmt.Errorf("scan node id: %w", errors.Join(err, rows.Close()))
		}
		nodeIDs = append(nodeIDs, nodeID)
	}
	return nodeIDs, nil
}
