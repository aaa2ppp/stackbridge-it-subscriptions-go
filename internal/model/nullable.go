package model

import (
	"encoding/json"
)

// Nullable позволяет различать отсутствие поля и значение null. Всегда используйте 'omitzero' в 'json' теге.
type Nullable[T any] struct {
	Value   T    `json:"-"`
	Defined bool `json:"-"`
	Valid   bool `json:"-"`
}

// MarshalJSON implements [json.Marshaler].
func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Value)
}

// UnmarshalJSON implements [json.Unmarshaler].
func (n *Nullable[T]) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		n.Defined = true
		return nil
	}
	if err := json.Unmarshal(b, &n.Value); err != nil {
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
