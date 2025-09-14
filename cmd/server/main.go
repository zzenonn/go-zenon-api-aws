package main

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/zzenonn/go-zenon-api-aws/internal/config"
	"github.com/zzenonn/go-zenon-api-aws/internal/factory"
	"github.com/zzenonn/go-zenon-api-aws/internal/logging"
)

// Instantiate and startup go app
func Run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	logging.InitLogger(cfg)
	log.Println("starting up the application")

	// Factory handles all dependency creation
	handlerFactory, err := factory.NewHandlerFactory(cfg)
	if err != nil {
		return err
	}

	if err := handlerFactory.MigrateUp(context.Background()); err != nil {
		return err
	}

	mainHandler := handlerFactory.CreateMainHandler()
	mainHandler.MapRoutes()

	return mainHandler.Serve()
}

func main() {

	log.Info("the server is up")

	if err := Run(); err != nil {
		log.Error(err)
	}
}
