package main

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/zzenonn/go-zenon-api-aws/internal/config"
	"github.com/zzenonn/go-zenon-api-aws/internal/logging"
	"github.com/zzenonn/go-zenon-api-aws/internal/repository/db"
	"github.com/zzenonn/go-zenon-api-aws/internal/repository/objectstore"
	"github.com/zzenonn/go-zenon-api-aws/internal/service"
	handlers "github.com/zzenonn/go-zenon-api-aws/internal/transport/http"
)

// Instantiate and startup go app
func Run() error {

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	logging.InitLogger(cfg)
	log.Println("starting up the application")

	dynamoDb, err := db.NewDatabase(cfg)

	if err != nil {
		log.Error("Failed to connect to the database")
		return err
	}

	if err := dynamoDb.MigrateDb(context.Background()); err != nil {
		log.Error("Failed to migrate the database")
		return err
	}

	userRepository := db.NewUserRepository(dynamoDb.Client, cfg.DynamoDBTable)
	userService := service.NewUserService(&userRepository)
	userHandler := handlers.NewUserHandler(userService, cfg)

	s3Store := objectstore.NewObjectStore(cfg)

	userProfileRepository := objectstore.NewUserProfileRepository(s3Store.Client, cfg.S3BucketName)
	userProfileService := service.NewUserProfileService(&userProfileRepository)
	userProfileHandler := handlers.NewUserProfileHandler(userProfileService, cfg)

	mainHandler := handlers.NewMainHandler(cfg)

	// httpHandler.AddHandler(commentHandler)
	mainHandler.AddHandler(userHandler)
	mainHandler.AddHandler(userProfileHandler)

	mainHandler.MapRoutes()

	if err := mainHandler.Serve(); err != nil {
		return err
	}

	return nil
}

func main() {

	log.Info("the server is up")

	if err := Run(); err != nil {
		log.Error(err)
	}
}
