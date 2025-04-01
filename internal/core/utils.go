package core

import (
	"github.com/SimonSchneider/goslu/sid"
)

func NewId() string {
	return sid.MustNewString(15)
}

func Coalesce[T comparable](a, b T) T {
	var zero T
	if a == zero {
		return b
	}
	return a
}
