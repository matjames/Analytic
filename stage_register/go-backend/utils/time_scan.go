package utils

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// FlexibleTime implements sql.Scanner and driver.Valuer so it can scan both
// time.Time and string/[]byte from the database (e.g. when the column is TEXT
// or the driver returns a string). This fixes "unsupported Scan, storing
// driver.Value type string into type *time.Time" errors.
type FlexibleTime struct {
	Time  time.Time
	Valid bool
}

// Scan implements sql.Scanner. Accepts time.Time, string, or []byte.
func (ft *FlexibleTime) Scan(value interface{}) error {
	if value == nil {
		ft.Valid = false
		ft.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		ft.Time = v
		ft.Valid = true
		return nil
	case []byte:
		return ft.parseString(string(v))
	case string:
		return ft.parseString(v)
	default:
		return fmt.Errorf("FlexibleTime: cannot scan type %T", value)
	}
}

// Value implements driver.Valuer.
func (ft FlexibleTime) Value() (driver.Value, error) {
	if !ft.Valid {
		return nil, nil
	}
	return ft.Time, nil
}

func (ft *FlexibleTime) parseString(s string) error {
	s = trimSpace(s)
	if s == "" {
		ft.Valid = false
		ft.Time = time.Time{}
		return nil
	}
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05.999999-07",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			ft.Time = t
			ft.Valid = true
			return nil
		}
	}
	ft.Valid = false
	ft.Time = time.Time{}
	return nil
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
