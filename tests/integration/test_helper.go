package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zzenonn/go-zenon-api-aws/internal/config"
	"github.com/zzenonn/go-zenon-api-aws/internal/factory"
)

// CreateTestConfig creates a test configuration using .env values
func CreateTestConfig(t *testing.T) *config.Config {
	cfg, err := config.LoadConfig()
	require.NoError(t, err)
	return cfg
}

// CreateTestHandlerFactory creates a handler factory for testing
func CreateTestHandlerFactory(t *testing.T) factory.TestableHandlerFactory {
	cfg := CreateTestConfig(t)

	handlerFactory, err := factory.NewHandlerFactory(cfg)
	require.NoError(t, err)

	return handlerFactory
}

// SetupTestServer creates a test server with migrations
func SetupTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	handlerFactory := CreateTestHandlerFactory(t)

	// Run migrations for test setup
	err := handlerFactory.MigrateUp(context.Background())
	require.NoError(t, err)

	mainHandler := handlerFactory.CreateMainHandler()
	mainHandler.MapRoutes()

	server := httptest.NewServer(mainHandler.Router)
	return server, server.Client()
}

// TeardownTestServer cleans up test server and database
func TeardownTestServer(server *httptest.Server) {
	server.Close()
	// Migrate down to clean up test data
	cfg, err := config.LoadConfig()
	if err != nil {
		return
	}
	handlerFactory, err := factory.NewHandlerFactory(cfg)
	if err != nil {
		return
	}
	handlerFactory.MigrateDown(context.Background())
}

// Test configuration constants
const (
	MaxRetries = 3
	BaseDelay  = 100 * time.Millisecond
)

// RetryWithBackoff executes a function with exponential backoff
func RetryWithBackoff(fn func() error) error {
	var err error
	for i := 0; i < MaxRetries; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if i < MaxRetries-1 {
			time.Sleep(BaseDelay * time.Duration(1<<i)) // Exponential backoff
		}
	}
	return err
}
