package model

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
)

// Nullable позволяет различать отсутствие поля и значение null. Всегда используйте 'omitzero' в 'json' теге.
type Nullable[T any] struct {
	sql.Null[T] `json:"-"`
	Defined     bool `json:"-"`
}

// MarshalJSON implements [json.Marshaler].
func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.V)
}

// UnmarshalJSON implements [json.Unmarshaler].
func (n *Nullable[T]) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		n.Defined = true
		return nil
	}
	if err := json.Unmarshal(b, &n.V); err != nil {
		return err
	}
	n.Defined = true
	n.Valid = true
	return nil
}

func (n Nullable[T]) IsZero() bool {
	return !n.Defined
}

var _ json.Unmarshaler = &Nullable[int]{}
var _ json.Marshaler = Nullable[int]{}
var _ sql.Scanner = &Nullable[int]{}
var _ driver.Valuer = Nullable[int]{}
