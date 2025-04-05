package db

import (
	"context"

	"cloud.google.com/go/firestore"
	log "github.com/sirupsen/logrus"
	"github.com/zzenonn/go-zenon-api-aws/internal/config"
)

type FirestoreDb struct {
	Client *firestore.Client
}

func NewDatabase(cfg *config.Config) (*FirestoreDb, error) {
	ctx := context.Background()

	var client *firestore.Client
	var err error

	client, err = firestore.NewClient(ctx, cfg.ProjectID)

	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}

	return &FirestoreDb{
		Client: client,
	}, nil
}
