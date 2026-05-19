package store

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
)

// NullInt64 is a JSON-friendly drop-in for sql.NullInt64. The stdlib
// version marshals to `{"Int64":5,"Valid":true}` which trips up every
// frontend that expects a plain `5` or `null`. We keep the same field
// names so existing `.Int64` / `.Valid` accessors keep working unchanged.
type NullInt64 struct {
	Int64 int64
	Valid bool
}

func (n NullInt64) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Int64)
}

func (n *NullInt64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Int64, n.Valid = 0, false
		return nil
	}
	var v int64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	n.Int64, n.Valid = v, true
	return nil
}

func (n *NullInt64) Scan(value interface{}) error {
	var ni sql.NullInt64
	if err := ni.Scan(value); err != nil {
		return err
	}
	n.Int64, n.Valid = ni.Int64, ni.Valid
	return nil
}

func (n NullInt64) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Int64, nil
}
