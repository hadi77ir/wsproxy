package utils

import (
	"fmt"
	"github.com/hadi77ir/go-logging"
	"go.uber.org/automaxprocs/maxprocs"
)

type UndoFunc func()

func SetMaxProcs(logger logging.Logger) UndoFunc {
	undo, err := maxprocs.Set(maxprocs.Logger(func(s string, i ...interface{}) {
		logger.Log(logging.DebugLevel, fmt.Sprintf(s, i...))
	}))

	if err != nil {
		logger.Log(logging.WarnLevel, "error setting GOMAXPROCS:", err)
		undo()
		return func() {}
	}
	return undo
}
