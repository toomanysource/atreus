package logX

import (
	"fmt"
	"io"
	"os"
	time2 "time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	kralog "github.com/go-kratos/kratos/v2/log"
	log "github.com/sirupsen/logrus"
)

type Log struct {
	Logger *log.Logger
}

func NewDefaultLogger() *Log {
	logger := log.New()
	logger.SetFormatter(&nested.Formatter{
		HideKeys:        false,
		NoColors:        true,
		ShowFullLevel:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		FieldsOrder:     []string{"service", "caller", "module", "msg"},
	})
	return &Log{
		Logger: logger,
	}
}

func (l *Log) Log(level kralog.Level, keyVal ...interface{}) error {
	entry := l.Logger.WithFields(log.Fields{})
	for i := 0; i < len(keyVal); i += 2 {
		key := keyVal[i]
		val := keyVal[i+1]
		entry = entry.WithField(key.(string), val)
	}

	switch level {
	case kralog.LevelDebug:
		entry.Debug()
	case kralog.LevelInfo:
		entry.Info()
	case kralog.LevelWarn:
		entry.Warn()
	case kralog.LevelError:
		entry.Error()
	default:
		entry.Print()
	}
	return nil
}

func (l *Log) SetOutput(w io.Writer) {
	l.Logger.SetOutput(w)
}

func (l *Log) SetLevel(level kralog.Level) {
	l.Logger.SetLevel(log.Level(level))
}

func (l *Log) FilePath(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
}

func (l *Log) SetTimeFileName(name string, flag bool) string {
	time := time2.Now()
	if flag {
		return fmt.Sprint(name, time.Format("2006-01-02 15:04:05"), ".log")
	}
	return fmt.Sprint(name, time.Format("2006-01-02"), ".log")
}
