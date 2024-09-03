package chore

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type Duration time.Duration

func ParseDuration(str string) (d Duration, err error) {
	dur, err := time.ParseDuration(str)
	if err != nil {
		return d, err
	}
	return Duration(dur), nil
}

func (d Duration) String() string {
	return time.Duration(d).String()
}

func (d Duration) Value() (driver.Value, error) {
	return d.String(), nil
}

func (d *Duration) Scan(src any) error {
	switch v := src.(type) {
	case int64:
		*d = Duration(v)
	case string:
		dur, err := time.ParseDuration(v)
		if err != nil {
			return err
		}
		*d = Duration(dur)
	default:
		return errors.New("unsupported type")
	}
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	dur, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
