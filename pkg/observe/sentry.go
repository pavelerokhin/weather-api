package observe

import (
	"encoding/json"
	"log"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"

	"go.uber.org/zap/zapcore"
)

const (
	_sentryMaxErrorDepth        int           = 9
	_sentryFlushTimeout         time.Duration = 5 * time.Second
	_sentryServerRequestTimeout time.Duration = 5 * time.Second
)

type SentryHook struct {
	appZone string
	appName string
	l       *Logger
}

func NewSentryHook(
	appZone, appName string,
	maxErrorDepth int,
	isDebug bool,
	dsn string,
) *SentryHook {
	if dsn == "" {
		log.Println("Stacktracer init error: no DSN")
	}
	if maxErrorDepth == 0 {
		maxErrorDepth = _sentryMaxErrorDepth
	}
	sentryTransport := sentry.NewHTTPTransport()
	sentryTransport.Timeout = _sentryServerRequestTimeout
	if err := sentry.Init(
		sentry.ClientOptions{
			AttachStacktrace: true,
			Debug:            isDebug,
			Dsn:              dsn,
			Environment:      appZone,
			MaxErrorDepth:    _sentryMaxErrorDepth,
			ServerName:       appName,
			Transport:        sentryTransport,
		}); err != nil {

		log.Println("Stacktracer init error: ", err.Error())
	}
	log.Println("Stacktracer init success")
	return &SentryHook{
		appZone: appZone,
		appName: appName,
	}
}

func (*SentryHook) mapLevel(zl zapcore.Level) sentry.Level {

	switch zl {

	case zapcore.DebugLevel, zapcore.InvalidLevel:
		return sentry.LevelDebug
	case zapcore.InfoLevel:
		return sentry.LevelInfo
	case zapcore.WarnLevel:
		return sentry.LevelWarning
	case zapcore.ErrorLevel:
		return sentry.LevelError
	case zapcore.FatalLevel, zapcore.PanicLevel:
		return sentry.LevelFatal

	}

	return sentry.LevelDebug
}

func (h *SentryHook) Write(p []byte) (n int, err error) {

	if h.appZone == "prod" || h.appZone == "dev" {
		type T struct {
			Level      string `json:"level"`
			AppName    string `json:"app_name"`
			AppEnv     string `json:"app_env"`
			CallerFile string `json:"caller_file"`
			CallerLine int    `json:"caller_line"`
			CallerFunc string `json:"caller_func"`
			Stack      string `json:"stack"`
			Message    string `json:"msg"`
			Error      string `json:"error"`
			Timestamp  string `json:"timestamp"`
		}
		t := T{}
		if err := json.Unmarshal(p, &t); err == nil {
			level, err := zapcore.ParseLevel(t.Level)
			if err == nil && len(t.Message) > 0 {
				timestamp, _ := time.ParseInLocation("2006-01-02T15-04-05.000", t.Timestamp, time.FixedZone("Europe/Moscow", 3*3600))

				switch level {

				case zapcore.ErrorLevel, zapcore.FatalLevel, zapcore.PanicLevel:
					event := sentry.NewEvent()
					event.Extra["AppName"] = h.appName
					event.Environment = h.appZone
					event.Level = h.mapLevel(level)
					event.Timestamp = timestamp
					event.Message = t.Message
					event.Extra["Error"] = t.Error
					event.Extra["CallerFile"] = t.CallerFile
					event.Extra["CallerLine"] = t.CallerLine
					event.Extra["CallerFunc"] = t.CallerFunc
					event.Extra["Stack"] = t.Stack
					event.Extra["TimeStamp"] = t.Timestamp
					for k, v := range event.Tags {
						if v == "" {
							delete(event.Tags, k)
						}
					}
					event.Exception = append(event.Exception, sentry.Exception{
						Type:       t.Message,
						Value:      t.Error,
						Stacktrace: sentry.NewStacktrace(),
					})
					sentry.CaptureEvent(event)
				}

			} else if err != nil {
				msg := errors.Wrap(err, "[SentryHook] parse zap level: ")
				if h.l != nil {
					h.l.Error(msg)
				} else {
					log.Println(msg.Error())
				}
			}

		} else {
			msg := errors.New("[SentryHook] json.Unmarshal data")
			if h.l != nil {
				h.l.Error(msg)
			} else {
				log.Println(msg.Error())
			}
		}

	}

	return len(p), nil
}

func (h *SentryHook) SetLogger(logger *Logger) {
	if logger != nil {
		h.l = logger
	}
}
