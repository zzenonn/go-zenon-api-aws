package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zzenonn/go-zenon-api-aws/internal/config"
	"github.com/zzenonn/go-zenon-api-aws/internal/factory"
)

// Test configuration constants
const (
	MaxRetries = 3
	BaseDelay  = 100 * time.Millisecond
)

// CreateTestConfig creates a test configuration using .env values
func CreateTestConfig(t *testing.T) *config.Config {
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load test configuration")
	return cfg
}

// CreateTestHandlerFactory creates a handler factory for testing
func CreateTestHandlerFactory(t *testing.T) factory.TestableHandlerFactory {
	cfg := CreateTestConfig(t)

	handlerFactory, err := factory.NewHandlerFactory(cfg)
	require.NoError(t, err, "Failed to create handler factory")

	return handlerFactory
}

// SetupTestServer creates a test server with migrations
func SetupTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	handlerFactory := CreateTestHandlerFactory(t)

	// Run migrations for test setup
	err := handlerFactory.MigrateUp(context.Background())
	require.NoError(t, err, "Failed to run database migrations")

	mainHandler := handlerFactory.CreateMainHandler()
	mainHandler.MapRoutes()

	server := httptest.NewServer(mainHandler.Router)
	return server, server.Client()
}

// TeardownTestServer cleans up test server and database
func TeardownTestServer(server *httptest.Server) {
	server.Close()

	// Clean up test data
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Warning: Failed to load config during teardown: %v\n", err)
		return
	}

	handlerFactory, err := factory.NewHandlerFactory(cfg)
	if err != nil {
		fmt.Printf("Warning: Failed to create handler factory during teardown: %v\n", err)
		return
	}

	err = handlerFactory.MigrateDown(context.Background())
	if err != nil {
		fmt.Printf("Warning: Failed to migrate down during teardown: %v\n", err)
	}
}

// RetryWithBackoff executes a function with exponential backoff
// Useful for handling eventual consistency in distributed systems
func RetryWithBackoff(fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < MaxRetries; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if attempt < MaxRetries-1 {
			delay := BaseDelay * time.Duration(1<<attempt) // Exponential backoff: 100ms, 200ms, 400ms
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("operation failed after %d attempts, last error: %w", MaxRetries, lastErr)
}
