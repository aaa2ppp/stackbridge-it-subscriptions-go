package model

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const DateLayout = "2006-01-02"

type Date struct {
	time.Time `json:"-"`
}

func (my *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")

	t, err := time.ParseInLocation(DateLayout, s, time.UTC)
	if err != nil {
		return fmt.Errorf("invalid Date format: %w", err)
	}

	my.Time = t
	return nil
}

func (my Date) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, "%q", my.Time.Format(DateLayout)), nil
}

func (my Date) Value() (driver.Value, error) {
	return my.Time.Format("2006-01-02"), nil
}

func (my *Date) Scan(value any) error {
	if v, ok := value.(time.Time); ok {
		v = v.UTC()
		my.Time = time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, time.UTC)
		return nil
	}
	return fmt.Errorf("cannot scan %T into Date", value)
}

var _ json.Unmarshaler = &Date{}
var _ json.Marshaler = Date{}
var _ sql.Scanner = &Date{}
var _ driver.Valuer = Date{}
