package core

import (
	"github.com/SimonSchneider/goslu/sid"
)

func NewId() string {
	return sid.MustNewString(15)
}
