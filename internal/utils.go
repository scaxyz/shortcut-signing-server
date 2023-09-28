package internal

import "github.com/rs/zerolog"

func NotImplemented(logger *zerolog.Logger) {
	logger.Warn().Caller(1).Msg("Not implemented")
}
