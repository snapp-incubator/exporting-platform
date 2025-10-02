package main

import (
	"context"
	"exporting_platform/configs"
	"exporting_platform/internal/application"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

var config *configs.Config

const defaultConfigPath = "./configs/config.yml"

func init() {
	var configPath = defaultConfigPath
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	fmt.Println("Reading config from:", configPath)
	cfg, err := configs.Load(configPath)
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
