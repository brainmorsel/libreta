package storage

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const schemaVersion = 1

const schema = `
CREATE TABLE schema_version (
	version INTEGER NOT NULL,
	created_at TEXT NOT NULL,
	PRIMARY KEY (version)
) STRICT;

CREATE TABLE node_content (
	hash TEXT NOT NULL,
	content BLOB NOT NULL,
	created_at TEXT NOT NULL,
	PRIMARY KEY (hash)
) STRICT;

CREATE TABLE node (
	fts_rowid INTEGER,
	id TEXT NOT NULL,
	name TEXT NOT NULL,
	content_hash TEXT NOT NULL,
	content_mimetype TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	deleted_at TEXT NULL,
	FOREIGN KEY (content_hash) REFERENCES node_content(hash),
	UNIQUE (id),
	PRIMARY KEY (fts_rowid)
) STRICT;

CREATE TABLE node_attribute (
	node_id TEXT NOT NULL,
	key TEXT NOT NULL,
	value TEXT NOT NULL,
	created_at TEXT NOT NULL,
	FOREIGN KEY (node_id) REFERENCES node(id),
	PRIMARY KEY (node_id, key, value)
) STRICT;

CREATE INDEX node_attribute_key_value_idx ON node_attribute(key, value);

CREATE TABLE edge (
	src_id TEXT NOT NULL,
	dst_id TEXT NOT NULL,
	relation TEXT NOT NULL,
	created_at TEXT NOT NULL,
	FOREIGN KEY (src_id) REFERENCES node(id),
	FOREIGN KEY (dst_id) REFERENCES node(id),
	PRIMARY KEY (src_id, dst_id, relation)
) STRICT;

CREATE INDEX edge_dst_rel_idx ON edge(dst_id, relation);

CREATE VIEW node_fts_view AS
	SELECT
		n.fts_rowid AS fts_rowid,
		n.id AS id,
		n.name AS name,
		c.content AS content
	FROM node AS n
	INNER JOIN node_content AS c ON n.content_hash = c.hash
;
CREATE VIRTUAL TABLE node_fts_idx USING fts5(id UNINDEXED, name, content, content='node_fts_view', content_rowid='fts_rowid', tokenize='trigram');
CREATE TRIGGER node_ai AFTER INSERT ON node BEGIN
	INSERT INTO node_fts_idx(rowid, name, content)
		SELECT new.fts_rowid, new.name, IIF(is_text_mimetype(new.content_mimetype), CAST(c.content AS TEXT), '')
		FROM node_content AS c
		WHERE hash = new.content_hash;
END;
CREATE TRIGGER node_ad AFTER DELETE ON node BEGIN
	INSERT INTO node_fts_idx(node_fts_idx, rowid, name, content)
		SELECT 'delete', old.fts_rowid, old.name, IIF(is_text_mimetype(old.content_mimetype), CAST(c.content AS TEXT), '')
		FROM node_content AS c
		WHERE hash = old.content_hash;
END;
CREATE TRIGGER node_au AFTER UPDATE ON node BEGIN
	INSERT INTO node_fts_idx(node_fts_idx, rowid, name, content)
		SELECT 'delete', old.fts_rowid, old.name, IIF(is_text_mimetype(old.content_mimetype), CAST(c.content AS TEXT), '')
		FROM node_content AS c
		WHERE hash = old.content_hash;
	INSERT INTO node_fts_idx(rowid, name, content)
		SELECT new.fts_rowid, new.name, IIF(is_text_mimetype(new.content_mimetype), CAST(c.content AS TEXT), '')
		FROM node_content AS c
		WHERE hash = new.content_hash;
END;
`

func (s *Storage) GetSchemaVersion(ctx context.Context) (int, error) {
	var version int
	err := s.readDB.Get(&version, `SELECT MAX(version) FROM schema_version`)
	if err != nil {
		if strings.Contains(err.Error(), "no such table: schema_version") {
			return 0, nil
		}
		return 0, err
	}
	return version, nil
}

func (s *Storage) CheckSchemaVersion(ctx context.Context, upgrade bool) error {
	version, err := s.GetSchemaVersion(ctx)
	if err != nil {
		return fmt.Errorf("get schema version: %w", err)
	}
	if version == schemaVersion {
		return nil
	}
	if version == 0 {
		// Freshly created db file.
		return s.setupNewDB(ctx)
	}
	if !upgrade {
		return fmt.Errorf("schema upgrade required from %d to %d", version, schemaVersion)
	}
	// TODO: upgrade db schema.
	return fmt.Errorf("schema upgrade required from %d to %d", version, schemaVersion)
}

func (s *Storage) setupNewDB(ctx context.Context) error {
	if _, err := s.writeDB.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	if _, err := s.writeDB.ExecContext(ctx, "INSERT INTO schema_version (version, created_at) VALUES ($1, $2)", schemaVersion, time.Now()); err != nil {
		return fmt.Errorf("update schema version: %w", err)
	}

	return nil
}
