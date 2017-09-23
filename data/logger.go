package data

import (
	"github.com/jackc/pgx"
	log "github.com/sirupsen/logrus"
)

// logger implementation for the PGX interface - this will wrap the Logurus log and map to it's
// levels.
type logger struct {
}

func (l *logger) Log(level pgx.LogLevel, msg string, data map[string]interface{}) {
	switch level {
	case pgx.LogLevelTrace:
	case pgx.LogLevelDebug:
		log.WithFields(data)
		log.Debug(msg)

	case pgx.LogLevelInfo:
		log.WithFields(data)
		log.Info(msg)

	case pgx.LogLevelWarn:
		log.WithFields(data)
		log.Warn(msg)

	case pgx.LogLevelError:
		log.WithFields(data)
		log.Error(msg)

	case pgx.LogLevelNone:
		log.WithFields(data)
		log.Info(msg)

	default:
		log.WithFields(data)
		log.Debug(msg)

	}
}
