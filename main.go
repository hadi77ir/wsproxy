package main

import (
	"context"
	"github.com/hadi77ir/go-logging/logrus"
	"github.com/hadi77ir/wsproxy/cmd"
	"log"
)

func main() {
	logger, err := logrus.New("wsproxy")
	if err != nil {
		log.Fatalf("failed to initialize logging facility:", err)
		return
	}
	_ = cmd.RootCmd.ExecuteContext(context.WithValue(context.Background(), "logger", logger))
}
