package duration

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Duration int64

const (
	toTimeDuration          = 24 * time.Hour
	Min            Duration = -Max
	Zero           Duration = 0
	Max            Duration = 1<<63 - 1
	Day            Duration = 1
	Week           Duration = 7 * Day
	Month          Duration = 30 * Day
	Year           Duration = 365 * Day
)

var (
	idents = [...]byte{'d', 'w', 'm', 'y'}
	vals   = [...]Duration{Day, Week, Month, Year}
)

func getMultiplier(byt byte) Duration {
	for i, id := range idents {
		if byt == id {
			return vals[i]
		}
	}
	return 0
}

func (d Duration) Zero() bool {
	return d == 0
}

func getIdent(mult Duration) byte {
	for i, val := range vals {
		if mult == val {
			return idents[i]
		}
	}
	return 0
}

func (d Duration) Prettify() string {
	switch {
	case d < -7*Day:
		return "Urgent"
	case d < 0:
		return "Overdue"
	case d < Day:
		return "Today"
	case d < 2*Day:
		return "Tomorrow"
	case d < 7*Day:
		return "This week"
	case d < 14*Day:
		return "Next week"
	case d < 30*Day:
		return "This month"
	default:
		return "Later"
	}
}

func ParseDuration(str string) (d Duration, err error) {
	if str == "" {
		return d, fmt.Errorf("invalid duration(%s): empty string", str)
	}
	str = strings.ToLower(str)
	bytes := []byte(str)
	var rootMult Duration = 1
	if bytes[0] == '-' {
		rootMult = -1
		bytes = bytes[1:]
	}
	start := 0
	for i := start; i < len(bytes); i++ {
		byt := bytes[i]
		if '0' < byt && byt < '9' {
			continue
		}
		mult := getMultiplier(byt)
		if mult == 0 {
			return d, fmt.Errorf("invalid duration(%s): no multi found", str)
		}
		if i == start {
			return d, fmt.Errorf("invalid duration(%s): no number before ident", str)
		}
		num, err := strconv.ParseInt(string(bytes[start:i]), 10, 64)
		if err != nil {
			return d, fmt.Errorf("invalid duration(%s): invalid number before ident: %s", str, bytes[start:i])
		}
		d += Duration(num) * mult
		start = i + 1
	}
	if start < len(bytes) {
		return d, fmt.Errorf("invalid duration(%s): no ident found", str)
	}
	return d * rootMult, nil
}

func (d Duration) String() string {
	if d == 0 {
		return "0"
	}
	var str string
	if d < 0 {
		str += "-"
		d = -d
	}
	for _, mult := range []Duration{Year, Month, Week, Day} {
		if d == 0 {
			break
		}
		num := int64(d / mult)
		if num == 0 {
			continue
		}
		d -= Duration(num) * mult
		ident := getIdent(mult)
		if ident == 0 {
			panic("invalid duration")
		}
		str += strconv.FormatInt(num, 10) + string(ident)
	}
	return str
}

func (d Duration) ToCompat() time.Duration {
	return time.Duration(d) * toTimeDuration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) (err error) {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	*d, err = ParseDuration(str)
	return err
}

func Sub(a, b time.Time) Duration {
	return Duration(a.Sub(b) / toTimeDuration)
}
