package storage

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
)

type Timestamp struct {
	time.Time
}

func (t Timestamp) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	return t.Time.Format(sqlite3.SQLiteTimestampFormats[0]), nil
}

func (t *Timestamp) Scan(value interface{}) error {
	var err error
	var timeVal time.Time
	switch v := value.(type) {
	case string:
		s := strings.TrimSuffix(v, "Z")
		for _, format := range sqlite3.SQLiteTimestampFormats {
			if timeVal, err = time.ParseInLocation(format, s, time.UTC); err == nil {
				t.Time = timeVal
				break
			}
		}
		if err != nil {
			return fmt.Errorf("failed to scan Timestamp: %w", err)
		}
	case nil:
		t.Time = time.Time{}
	default:
		return fmt.Errorf("failed to scan Timestamp: unsupported column type")
	}
	return nil
}
