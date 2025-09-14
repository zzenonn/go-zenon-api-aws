package factory

import (
	"context"

	"github.com/zzenonn/go-zenon-api-aws/internal/config"
	"github.com/zzenonn/go-zenon-api-aws/internal/repository/db"
	"github.com/zzenonn/go-zenon-api-aws/internal/repository/objectstore"
	"github.com/zzenonn/go-zenon-api-aws/internal/service"
	handlers "github.com/zzenonn/go-zenon-api-aws/internal/transport/http"
)

type TestableHandlerFactory interface {
	CreateMainHandler() *handlers.MainHandler
	MigrateUp(ctx context.Context) error
	MigrateDown(ctx context.Context) error
}

type HandlerFactory struct {
	cfg *config.Config
	db  *db.DynamoDb
	s3  *objectstore.S3Store
}

func (f *HandlerFactory) MigrateUp(ctx context.Context) error {
	return f.db.MigrateDb(ctx)
}

func (f *HandlerFactory) MigrateDown(ctx context.Context) error {
	return f.db.MigrateDown(ctx)
}

func NewHandlerFactory(cfg *config.Config) (*HandlerFactory, error) {
	// Initialize shared dependencies once
	dynamoDb, err := db.NewDatabase(cfg)
	if err != nil {
		return nil, err
	}

	s3Store := objectstore.NewObjectStore(cfg)

	return &HandlerFactory{
		cfg: cfg,
		db:  dynamoDb,
		s3:  s3Store,
	}, nil
}

func (f *HandlerFactory) CreateUserHandler() *handlers.UserHandler {
	userRepo := db.NewUserRepository(f.db.Client, f.cfg.DynamoDBTable)
	profileRepo := objectstore.NewUserProfileRepository(f.s3.Client, f.cfg.S3BucketName)
	userService := service.NewUserService(&userRepo, &profileRepo)
	return handlers.NewUserHandler(userService, f.cfg)
}

func (f *HandlerFactory) CreateMainHandler() *handlers.MainHandler {
	mainHandler := handlers.NewMainHandler(f.cfg)

	// Auto-register all handlers
	mainHandler.AddHandler(f.CreateUserHandler())

	return mainHandler
}