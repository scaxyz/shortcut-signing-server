package internal

import (
	"github.com/rs/zerolog"
)

var logger = zerolog.Nop()

func SetLogger(newLogger zerolog.Logger) {
	logger = newLogger
}
