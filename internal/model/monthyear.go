package model

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

const MonthYearLayout = "01-2006"

type MonthYear struct {
	time.Time `json:"-"`
}

func (my *MonthYear) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		return nil
	}

	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}

	t, err := time.ParseInLocation(MonthYearLayout, s, time.UTC)
	if err != nil {
		return fmt.Errorf("invalid MonthYear format: %w", err)
	}

	my.Time = t
	return nil
}

func (my MonthYear) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, "%q", my.Format(MonthYearLayout)), nil
}

func (my MonthYear) Value() (driver.Value, error) {
	return my.Format("2006-01-02"), nil
}

func (my *MonthYear) Scan(value any) error {
	if v, ok := value.(time.Time); ok {
		v = v.UTC()
		my.Time = time.Date(v.Year(), v.Month(), 1, 0, 0, 0, 0, time.UTC)
		return nil
	}
	return fmt.Errorf("cannot scan %T into MonthYear", value)
}

func (my MonthYear) IsZero() bool {
	return my.Time.IsZero()
}

func (my MonthYear) Sub(other MonthYear) int {
	return (my.Year()-other.Year())*12 + int(my.Month()) - int(other.Month())
}

var _ json.Unmarshaler = &MonthYear{}
var _ json.Marshaler = MonthYear{}
var _ sql.Scanner = &MonthYear{}
var _ driver.Valuer = MonthYear{}
