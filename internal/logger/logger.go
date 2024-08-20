// Модуль logger предоставляет для всего приложения (глобально) настроенный логгер.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log настроенный логгер доступный по всему приложению.
var Log *zap.SugaredLogger = zap.NewNop().Sugar()

// Initialize инициализирует логгер согласно переданным настройкам.
func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zl, err := cfg.Build()
	if err != nil {
		return err
	}
	Log = zl.Sugar()
	return nil
}
