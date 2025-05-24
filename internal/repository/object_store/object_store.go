package objectstore

import (
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	log "github.com/sirupsen/logrus"
	"github.com/zzenonn/go-zenon-api-aws/internal/config"
)

type S3Store struct {
	Client        *s3.Client
	TaggingClient *resourcegroupstaggingapi.Client
}

func NewObjectStore(cfg *config.Config) (*S3Store, error) {
	client := s3.NewFromConfig(cfg.AwsConfig)
	if client == nil {
		log.Fatal("Failed to create S3 client")
	}

	taggingClient := resourcegroupstaggingapi.NewFromConfig(cfg.AwsConfig)
	if taggingClient == nil {
		log.Fatal("Failed to create Resource Groups Tagging API client")
	}

	return &S3Store{
		Client:        client,
		TaggingClient: taggingClient,
	}, nil
}
