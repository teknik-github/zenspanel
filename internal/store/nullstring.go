package store

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
)

// NullString is a JSON-friendly drop-in for sql.NullString. The stdlib
// version marshals to `{"String":"foo","Valid":true}` — we want plain
// `"foo"` or `null` instead.
type NullString struct {
	String string
	Valid  bool
}

func (n NullString) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.String)
}

func (n *NullString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.String, n.Valid = "", false
		return nil
	}
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	n.String, n.Valid = v, true
	return nil
}

func (n *NullString) Scan(value interface{}) error {
	var ns sql.NullString
	if err := ns.Scan(value); err != nil {
		return err
	}
	n.String, n.Valid = ns.String, ns.Valid
	return nil
}

func (n NullString) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.String, nil
}
