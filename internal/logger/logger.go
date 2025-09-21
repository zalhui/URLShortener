package logger

import (
	"fmt"

	"go.uber.org/zap"
)

var Sugar zap.SugaredLogger

func Init() error {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}
	Sugar = *logger.Sugar()

	return nil
}
