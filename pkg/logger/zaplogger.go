package logger

import (
	"io"
	"os"
	"runtime"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	appEnv  string
	appName string
	l       *zap.Logger
}

func NewZapLogger(appName string, writers ...io.Writer) *Logger {

	var multiWriters []zapcore.WriteSyncer

	cfg := zap.NewProductionEncoderConfig()

	cfg.EncodeTime = timeEncoder("2006-01-02T15-04-05.000", time.FixedZone("Europe/Rome", 3*3600))
	cfg.TimeKey = "timestamp"

	if len(writers) == 0 {
		multiWriters = append(multiWriters, os.Stdout)
	} else {
		for _, writer := range writers {
			multiWriters = append(multiWriters, zapcore.AddSync(writer))
		}
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		zapcore.NewMultiWriteSyncer(multiWriters...),
		zapcore.DebugLevel,
	)

	return &Logger{
		appName: appName,
		l:       zap.New(core),
	}
}

func (l *Logger) Stop() (err error) {
	if err = l.l.Sync(); err != nil {
		return
	}
	return
}

func (l *Logger) Error(err error, fields ...map[string]any) {
	file, line, funcName := getRuntimeParams()
	zapFields := []zapcore.Field{}
	if len(fields) > 0 {
		zapFields = mapToZapFields(fields[0])
	}
	l.l.WithOptions(zap.Fields(zapFields...)).Error(
		err.Error(),
		zap.String("app_zone", l.appEnv),
		zap.String("app_name", l.appName),
		zap.String("error", err.Error()),
		zap.String("caller_file", file),
		zap.Int("caller_line", line),
		zap.String("caller_func", funcName),
		zap.Stack("stack"),
	)
}

func (l *Logger) Info(msg string, fields ...map[string]any) {
	file, line, funcName := getRuntimeParams()
	zapFields := []zapcore.Field{}
	if len(fields) > 0 {
		zapFields = mapToZapFields(fields[0])
	}
	l.l.WithOptions(zap.Fields(zapFields...)).Info(
		msg,
		zap.String("app_zone", l.appEnv),
		zap.String("app_name", l.appName),
		zap.Any("caller_file", file),
		zap.Any("caller_line", line),
		zap.Any("caller_func", funcName))
}

func (l *Logger) Warning(msg string, fields ...map[string]any) {
	file, line, funcName := getRuntimeParams()
	zapFields := []zapcore.Field{}
	if len(fields) > 0 {
		zapFields = mapToZapFields(fields[0])
	}
	l.l.WithOptions(zap.Fields(zapFields...)).Warn(
		msg,
		zap.String("app_zone", l.appEnv),
		zap.String("app_name", l.appName),
		zap.Any("caller_file", file),
		zap.Any("caller_line", line),
		zap.Any("caller_func", funcName))

}

func (l *Logger) Debug(msg string, fields ...map[string]any) {
	file, line, funcName := getRuntimeParams()
	zapFields := []zapcore.Field{}
	if len(fields) > 0 {
		zapFields = mapToZapFields(fields[0])
	}
	l.l.WithOptions(zap.Fields(zapFields...)).Debug(
		msg,
		zap.String("app_zone", l.appEnv),
		zap.String("app_name", l.appName),
		zap.Any("caller_file", file),
		zap.Any("caller_line", line),
		zap.Any("caller_func", funcName))
}

func (l *Logger) Fatal(msg string, fields ...map[string]any) {
	file, line, funcName := getRuntimeParams()
	zapFields := []zapcore.Field{}
	if len(fields) > 0 {
		zapFields = mapToZapFields(fields[0])
	}
	l.l.WithOptions(zap.Fields(zapFields...)).Fatal(
		msg,
		zap.String("app_zone", l.appEnv),
		zap.String("app_name", l.appName),
		zap.Any("caller_file", file),
		zap.Any("caller_line", line),
		zap.Any("caller_func", funcName))
}

func (l *Logger) Log(keyvals ...any) error {
	l.l.Info("", toZapFields(keyvals)...)

	return nil
}

func toZapFields(keyvals []any) []zap.Field {
	fields := make([]zap.Field, 0, len(keyvals)/2)

	for i := 0; i < len(keyvals); i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			key = "invalid-key"
		}

		fields = append(fields, zap.Any(key, keyvals[i+1]))
	}

	return fields
}

func mapToZapFields(data map[string]any) []zap.Field {
	zapFields := make([]zap.Field, 0, len(data))

	for k, v := range data {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	return zapFields
}

func getRuntimeParams() (file string, line int, funcName string) {
	var ok bool
	var pc uintptr
	pc, file, line, ok = runtime.Caller(2)
	if !ok {
		file = "not_defined"
		line = 0
		funcName = "not_defined"
	} else {
		funcName = runtime.FuncForPC(pc).Name()
	}
	return

}

func timeEncoder(layout string, location *time.Location) func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		t = t.In(location)
		type appendTimeEncoder interface {
			AppendTimeLayout(time.Time, string)
		}
		if enc, ok := enc.(appendTimeEncoder); ok {
			enc.AppendTimeLayout(t, layout)
			return
		}
		enc.AppendString(t.Format(layout))
	}
}
