package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func init() {
	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: true,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "at",
			LevelKey:       "level",
			CallerKey:      "from",
			MessageKey:     "msg",
			StacktraceKey:  "trace",
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
			EncodeCaller:   zapcore.ShortCallerEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
		},
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		OutputPaths:       []string{"stdout", "./logs/logs.json"},
		ErrorOutputPaths:  []string{"stderr"},
	}

	Log = zap.Must(config.Build())
}
