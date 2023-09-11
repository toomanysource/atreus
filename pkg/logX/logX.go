package logX

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	LogMaxSize    = 10
	LogMaxBackups = 5
	LogMaxAge     = 30
	SkipCaller    = 2
)

var _ log.Logger = (*ZapLogger)(nil)

type ZapLogger struct {
	log  *zap.Logger
	Sync func() error
}

// Logger 配置zap日志,将zap日志库引入
func Logger() log.Logger {
	timeEncoder := func(t time.Time, e zapcore.PrimitiveArrayEncoder) {
		var builder strings.Builder
		builder.WriteString("[")
		builder.WriteString(t.Format("2006-01-02 15:04:05.000"))
		builder.WriteString("]")
		e.AppendString(builder.String())
	}
	levelEncoder := func(l zapcore.Level, e zapcore.PrimitiveArrayEncoder) {
		var builder strings.Builder
		builder.WriteString("[")
		builder.WriteString(l.CapitalString())
		builder.WriteString("]")
		e.AppendString(builder.String())
	}
	encoder := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		StacktraceKey:  "stack",
		EncodeTime:     timeEncoder,
		EncodeLevel:    levelEncoder,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}
	return NewZapLogger(
		encoder,
		zap.NewAtomicLevelAt(zapcore.DebugLevel),
		zap.AddStacktrace(
			zap.NewAtomicLevelAt(zapcore.ErrorLevel)),
		zap.AddCaller(),
		zap.AddCallerSkip(SkipCaller),
		zap.Development(),
	)
}

// 日志自动切割，采用 lumberjack 实现的
func getLogWriter() zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   "../../../../logs/zap.log",
		MaxSize:    LogMaxSize,    // 日志的最大大小（M）
		MaxBackups: LogMaxBackups, // 日志的最大保存数量
		MaxAge:     LogMaxAge,     // 日志文件存储最大天数
		Compress:   false,         // 是否执行压缩
	}
	return zapcore.AddSync(lumberJackLogger)
}

// NewZapLogger return a zap logger.
func NewZapLogger(encoder zapcore.EncoderConfig, level zap.AtomicLevel, opts ...zap.Option) *ZapLogger {
	// 日志切割
	writeSyncer := getLogWriter()
	// 设置日志级别
	level.SetLevel(zap.InfoLevel)
	core := zapcore.NewCore(
		// 编码器配置
		zapcore.NewConsoleEncoder(encoder),
		// 打印到控制台和文件
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(writeSyncer)),
		// 日志级别
		level,
	)
	zapLogger := zap.New(core, opts...)
	return &ZapLogger{log: zapLogger, Sync: zapLogger.Sync}
}

// Log 实现log接口
func (l *ZapLogger) Log(level log.Level, keyvals ...interface{}) error {
	if len(keyvals) == 0 || len(keyvals)%2 != 0 {
		l.log.Warn(fmt.Sprint("Keyvalues must appear in pairs: ", keyvals))
		return nil
	}
	var data []zap.Field
	for i := 0; i < len(keyvals); i += 2 {
		data = append(data, zap.Any(fmt.Sprint(keyvals[i]), keyvals[i+1]))
	}

	switch level {
	case log.LevelDebug:
		l.log.Debug("", data...)
	case log.LevelInfo:
		l.log.Info("", data...)
	case log.LevelWarn:
		l.log.Warn("", data...)
	case log.LevelError:
		l.log.Error("", data...)
	case log.LevelFatal:
		l.log.Fatal("", data...)
	}
	return nil
}
