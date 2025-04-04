package integration

import (
	"context"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// SecretsManagerService defines the interface for secrets fetching
type SecretsManagerService interface {
	GetSecretValue(ctx context.Context, secretName string) (string, error)
}

// GoogleSecretsManagerService is an implementation of SecretsManagerService using Google Cloud Secret Manager
type GoogleSecretsManagerService struct {
	Client *secretmanager.Client
}

// NewGoogleSecretsManagerService creates a new GoogleSecretsManagerService
func NewGoogleSecretsManagerService() (*GoogleSecretsManagerService, error) {
	// Initialize the Google Secret Manager client
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)

	client, err = secretmanager.NewClient(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create Google Secret Manager client: %w", err)
	}

	return &GoogleSecretsManagerService{
		Client: client,
	}, nil
}

// GetSecretValue retrieves the secret from Google Secret Manager
func (s *GoogleSecretsManagerService) GetSecretValue(ctx context.Context, secretPath string) (string, error) {
	// Construct the request to access the secret version
	secretRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretPath, // The resource name of the secret version
	}

	// Access the secret version
	result, err := s.Client.AccessSecretVersion(ctx, secretRequest)
	if err != nil {
		return "", fmt.Errorf("failed to access secret %s: %w", secretPath, err)
	}

	// Return the secret payload as a string
	return string(result.Payload.Data), nil
}
