package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var ErrNoRecord = errors.New("record not found")

type Storage struct {
	DataDir string

	logger *slog.Logger

	writeDB *sqlx.DB
	readDB  *sqlx.DB

	nodeIDMu      sync.Mutex
	nodeIDLast    string
	nodeIDCounter int
}

func NewStorage(logger *slog.Logger, dataDir string) (*Storage, error) {
	storage := &Storage{
		DataDir: dataDir,
		logger:  logger,
	}
	return storage, nil
}

func (s *Storage) Open(ctx context.Context, upgrade bool) error {
	if err := s.openWrite(ctx); err != nil {
		return fmt.Errorf("write conn: %w", err)
	}
	if err := s.openRead(ctx); err != nil {
		return fmt.Errorf("read conn: %w", err)
	}
	if err := s.CheckSchemaVersion(ctx, upgrade); err != nil {
		return fmt.Errorf("check schema version: %w", err)
	}
	return nil
}

func (s *Storage) Close() error {
	wCloseErr := s.writeDB.Close()
	rCloseErr := s.readDB.Close()
	return errors.Join(wCloseErr, rCloseErr)
}

func (s *Storage) connURI(mode string) string {
	connURI := s.DataDir + "/data.db"
	switch mode {
	case "read":
		connURI += "?mode=r"
	case "write":
		connURI += "?mode=rw"
	case "create":
		connURI += "?mode=rwc"
	default:
		panic(fmt.Errorf("invalid db open mode: %s", mode))
	}
	connURI += "&_txlock=immediate"
	return connURI
}

func (s *Storage) openWrite(ctx context.Context) (err error) {
	connURI := s.connURI("create")
	s.writeDB, err = sqlx.Open(sqliteDriverName, connURI)
	if err != nil {
		return fmt.Errorf("sqlite open %q: %w", connURI, err)
	}
	if err := s.writeDB.PingContext(ctx); err != nil {
		return fmt.Errorf("sqlite open %q: %w", connURI, err)
	}
	s.writeDB.SetMaxOpenConns(1)
	s.logger.Debug("sqlite open write connection", slog.String("conn_uri", connURI))

	s.writeDB.ExecContext(ctx, "PRAGMA journal_mode = WAL;")
	s.writeDB.ExecContext(ctx, "PRAGMA synchronous = NORMAL;")
	s.writeDB.ExecContext(ctx, "PRAGMA cache_size = 10000;") // pages
	s.writeDB.ExecContext(ctx, "PRAGMA foreign_keys = true;")
	s.writeDB.ExecContext(ctx, "PRAGMA busy_timeout = 5000;") // ms

	return nil
}

func (s *Storage) openRead(ctx context.Context) (err error) {
	connURI := s.connURI("read")
	s.readDB, err = sqlx.Open(sqliteDriverName, connURI)
	if err != nil {
		return fmt.Errorf("sqlite open %q: %w", connURI, err)
	}
	if err := s.readDB.PingContext(ctx); err != nil {
		return fmt.Errorf("sqlite open %q: %w", connURI, err)
	}
	s.readDB.SetMaxOpenConns(max(4, runtime.NumCPU()))
	s.logger.Debug("sqlite open read connection", slog.String("conn_uri", connURI))

	s.readDB.ExecContext(ctx, "PRAGMA journal_mode = WAL;")
	s.readDB.ExecContext(ctx, "PRAGMA synchronous = NORMAL;")
	s.readDB.ExecContext(ctx, "PRAGMA cache_size = 10000;") // pages
	s.readDB.ExecContext(ctx, "PRAGMA foreign_keys = true;")
	s.readDB.ExecContext(ctx, "PRAGMA busy_timeout = 5000;") // ms

	return nil
}
