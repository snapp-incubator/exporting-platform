package main

import (
	"context"
	"exporting_platform/configs"
	"exporting_platform/internal/application"
	log "github.com/sirupsen/logrus"
	"time"
)

var config *configs.Config

func init() {
	cfg, err := configs.Load("./configs/config.yml")
	if err != nil {
		log.WithError(err).Fatal("Error while loading configurations")
	}
	config = cfg
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat:  time.RFC3339,
		DisableTimestamp: false,
		FieldMap:         nil,
		CallerPrettyfier: nil,
	})
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	app, err := application.NewApplication(config)
	if err != nil {
		log.Fatal(err)
	}
	app.Run(ctx)
	cancel()
}
