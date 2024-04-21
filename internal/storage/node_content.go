package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/mattn/go-sqlite3"
)

type nodeContentRow struct {
	Hash      string    `db:"hash"`
	Content   []byte    `db:"content"`
	CreatedAt Timestamp `db:"created_at"`
}

func (s *Storage) NodeContentLoad(ctx context.Context, hash string) (r io.Reader, err error) {
	// TODO: replace with Blob I/O, when https://github.com/mattn/go-sqlite3/issues/239 resolved.
	var row nodeContentRow
	err = s.readDB.GetContext(ctx, &row, `SELECT hash, content, created_at FROM node_content WHERE hash = $1`, hash)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrNoRecord
	case err != nil:
		return nil, fmt.Errorf("select: %w", err)
	}
	return bytes.NewReader(row.Content), nil
}

func (s *Storage) NodeContentSave(ctx context.Context, r io.Reader) (string, error) {
	w := sha256.New()
	tee := io.TeeReader(r, w)

	now := time.Now()
	tmpHash := "tmp:" + now.Format("2006-01-02 15:04:05.999999999-07:00")

	txCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	tx, err := s.writeDB.BeginTxx(txCtx, &sql.TxOptions{})
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}

	// TODO: replace with Blob I/O, when https://github.com/mattn/go-sqlite3/issues/239 resolved.
	buf, err := io.ReadAll(tee)
	if err != nil {
		return "", fmt.Errorf("read content: %w", err)
	}
	_, err = tx.ExecContext(
		txCtx,
		`INSERT INTO node_content(hash, content, created_at) VALUES ($1, $2, $3)`,
		tmpHash, buf, now,
	)
	if err != nil {
		return "", fmt.Errorf("insert content: %w", err)
	}

	hash := hex.EncodeToString(w.Sum(nil))
	_, err = tx.ExecContext(
		txCtx, `UPDATE node_content SET hash = $1 WHERE hash = $2`, hash, tmpHash,
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && errors.Is(sqliteErr.Code, sqlite3.ErrConstraint) {
			return hash, nil
		}
		return "", fmt.Errorf("update hash: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit: %w", err)
	}

	return hash, nil
}
