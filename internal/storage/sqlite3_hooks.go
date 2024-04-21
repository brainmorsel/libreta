package storage

import (
	"database/sql"
	"strings"

	"github.com/mattn/go-sqlite3"
)

const sqliteDriverName = "sqlite3_custom"

func init() {
	sql.Register(sqliteDriverName, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			if err := conn.RegisterFunc("is_text_mimetype", isTextMimetype, true); err != nil {
				return err
			}
			return nil
		},
	})
}

func isTextMimetype(mimetype string) bool {
	switch {
	case strings.HasPrefix(mimetype, "text/"):
		return true
	case mimetype == "application/json":
		return true
	default:
		return false
	}
}
