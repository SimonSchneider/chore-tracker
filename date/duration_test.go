package date

import "testing"

func TestParseAndPrintDuration(t *testing.T) {
	tests := []struct {
		s string
		e string
		d Duration
	}{
		{s: "1d", d: Day},
		{s: "1w", d: Week},
		{s: "1m", d: Month},
		{s: "-1y", d: -1 * Year},
		{s: "1D1W1M1Y", e: "1y1m1w1d", d: Year + Month + Week + Day},
		{s: "1d2d", e: "3d", d: 3 * Day},
		{s: "88d", e: "2m4w", d: 88 * Day},
		{s: "14d", e: "2w", d: 2 * Week},
		{s: "1y2m", d: Year + 2*Month},
	}
	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			d, err := ParseDuration(test.s)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d != test.d {
				t.Fatalf("expected %d, got %d", test.d, d)
			}
			exp := test.s
			if test.e != "" {
				exp = test.e
			}
			if d.String() != exp {
				t.Fatalf("expected %v, got %v", exp, d.String())
			}
		})
	}
}

func TestInvalidStringParsing(t *testing.T) {
	tests := []string{
		"",
		"0",
		"1",
		"1d2",
	}
	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			d, err := ParseDuration(test)
			if err == nil {
				t.Fatalf("expected error, got nil err and dur %s", d)
			}
		})
	}
}
