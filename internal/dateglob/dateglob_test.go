package dateglob_test

import (
	"github.com/SimonSchneider/chore-tracker/internal/dateglob"
	"github.com/SimonSchneider/goslu/date"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  dateglob.DateGlob
		isErr bool
	}{
		{input: "*-10-01", want: dateglob.New("01", "10", "*")},
		{input: "01-{-*", isErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := dateglob.Parse(tt.input)
			if err != nil && !tt.isErr {
				t.Fatalf("unexpected error: %s", err)
			}
			if err == nil && tt.isErr {
				t.Fatalf("expected error, got none")
			} else {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Parse() = %v, want %v", got, tt.want)
				}

			}
		})
	}
}

func TestDateGlob_NextMatch(t *testing.T) {
	day, err := date.ParseDate("2021-01-02")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tests := []struct {
		dg   string
		want string
	}{
		{dg: "*-1-1", want: "2022-01-01"},
		{dg: "*-*-1", want: "2021-02-01"},
		{dg: "*-2-*", want: "2021-02-01"},
		{dg: "*-1-4", want: "2021-01-04"},
	}
	for _, tt := range tests {
		t.Run(tt.dg, func(t *testing.T) {
			p, err := dateglob.Parse(tt.dg)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			got, err := p.NextMatch(day)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.String() != tt.want {
				t.Errorf("NextMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}
