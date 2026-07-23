package date

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

const MonthYearLayout = "01-2006"

type MonthYear struct {
	time.Time `json:"-"`
}

func (my *MonthYear) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")

	t, err := time.ParseInLocation(MonthYearLayout, s, time.UTC)
	if err != nil {
		return fmt.Errorf("invalid MonthYear format: %w", err)
	}

	my.Time = t
	return nil
}

func (my MonthYear) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, "%q", my.Time.Format(MonthYearLayout)), nil
}

func (my MonthYear) Value() (driver.Value, error) {
	return my.Time.Format("2006-01-02"), nil
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
