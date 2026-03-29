/**
 * File: logger.go
 * Project: logger
 * Created Date: 2026‑03‑28T21:41:29.2929+01:00
 * Author: ZanderCodes (Julian Zander) <admin@zandercodes.com>
 *
 * Last Modified: 2026‑03‑28T22:03:01.011+01:00
 * Modified By: ZanderCodes (Julian Zander) <admin@zandercodes.com>
 *
 * Copyright © 2026 ZanderCodes (Julian Zander). All rights reserved.
 */

package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

type Log struct {
	zerolog.Logger
}

func NewLogger() Log {
	zl := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Caller().Logger()
	zl = zl.Level(zerolog.InfoLevel)
	return Log{zl}
}

func NewLoggerWithLevel(level zerolog.Level) Log {
	zl := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Caller().Logger()
	zl = zl.Level(level)
	return Log{zl}
}
