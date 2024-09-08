package date

import (
	"encoding/json"
	"time"
)

const timeFrac = 24 * 60 * 60

type Date int64

func (d Date) IsZero() bool {
	return d == 0
}

func (d Date) ToStdTime() time.Time {
	return time.Unix(int64(d)*timeFrac, 0)
}

func FromTime(t time.Time) Date {
	return Date(t.Unix() / timeFrac)
}

func (d Date) Add(days Duration) Date {
	return d + Date(days)
}

func (d Date) Sub(d2 Date) Duration {
	return Duration(d - d2)
}

func (d Date) Before(d2 Date) bool {
	return d < d2
}

func (d Date) After(d2 Date) bool {
	return d > d2
}

func Today() Date {
	return FromTime(time.Now())
}

func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.String() + `"`), nil
}

func (d *Date) UnmarshalJSON(b []byte) (err error) {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	*d, err = ParseDate(str)
	return err
}

func (d Date) String() string {
	return d.ToStdTime().Format("2006-01-02")
}

func ParseDate(str string) (Date, error) {
	t, err := time.Parse("2006-01-02", str)
	if err != nil {
		return 0, err
	}
	return FromTime(t), nil
}
