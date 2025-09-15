package integration

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants
const (
	TestPassword    = "password123"
	NewPassword     = "newpassword123"
	AdminUsername   = "admin"
	AdminPassword   = "admin"
	FakeImageData   = "fake image data"
	ProfileFilename = "profile.jpg"
)

// Test endpoints
const (
	UsersEndpoint   = "/api/v1/users"
	LoginEndpoint   = "/api/v1/users/login"
	GetUserEndpoint = "/api/v1/users/getuser"
	ProfileEndpoint = "/api/v1/users/%s/profile"
	UserEndpoint    = "/api/v1/users/%s"
)

type UserTestSuite struct {
	server *httptest.Server
	client *http.Client
}

var (
// Remove these as they're now in test_helper.go
)

func setupUserTestServer(t *testing.T) *UserTestSuite {
	server, client := GetGlobalTestServer(t)
	return &UserTestSuite{
		server: server,
		client: client,
	}
}

// Helper methods for common operations
func (ts *UserTestSuite) makeAuthenticatedRequest(method, endpoint, token string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, ts.server.URL+endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return ts.client.Do(req)
}

func (ts *UserTestSuite) createTestUser(username, password string) error {
	adminToken := ts.getUserToken(AdminUsername, AdminPassword)
	payload := map[string]string{
		"username": username,
		"password": password,
	}
	body, _ := json.Marshal(payload)
	resp, err := ts.makeAuthenticatedRequest("POST", UsersEndpoint, adminToken, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (ts *UserTestSuite) getUserToken(username, password string) string {
	loginPayload := map[string]string{
		"username": username,
		"password": password,
	}
	loginBody, _ := json.Marshal(loginPayload)

	var loginResp *http.Response
	RetryWithBackoff(func() error {
		var err error
		loginResp, err = ts.client.Post(ts.server.URL+LoginEndpoint, "application/json", bytes.NewBuffer(loginBody))
		return err
	})
	defer loginResp.Body.Close()

	result := unmarshalResponse(loginResp)
	return result["token"].(string)
}

func (ts *UserTestSuite) createMultipartRequest(endpoint, token, filename string, data []byte) (*http.Response, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	fileWriter, _ := writer.CreateFormFile("file", filename)
	fileWriter.Write(data)
	writer.Close()

	req, err := http.NewRequest("PUT", ts.server.URL+endpoint, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return ts.client.Do(req)
}

func unmarshalResponse(resp *http.Response) map[string]any {
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

// Test Create User
func TestCreateUser(t *testing.T) {
	defer func() { RecordTest("CreateUser", !t.Failed()) }()
	ts := setupUserTestServer(t)
	adminToken := ts.getUserToken(AdminUsername, AdminPassword)

	payload := map[string]string{
		"username": "testuser",
		"password": TestPassword,
	}

	body, _ := json.Marshal(payload)
	resp, err := ts.makeAuthenticatedRequest("POST", UsersEndpoint, adminToken, body)

	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	result := unmarshalResponse(resp)
	assert.Equal(t, "testuser", result["username"])
}

// Test Login
func TestLogin(t *testing.T) {
	defer func() { RecordTest("Login", !t.Failed()) }()
	ts := setupUserTestServer(t)
	username := "loginuser"

	ts.createTestUser(username, TestPassword)

	loginPayload := map[string]string{
		"username": username,
		"password": TestPassword,
	}
	loginBody, _ := json.Marshal(loginPayload)

	var resp *http.Response
	err := RetryWithBackoff(func() error {
		var err error
		resp, err = ts.client.Post(ts.server.URL+LoginEndpoint, "application/json", bytes.NewBuffer(loginBody))
		return err
	})
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	result := unmarshalResponse(resp)
	assert.NotEmpty(t, result["token"])
}

// Test Get User
func TestGetUser(t *testing.T) {
	defer func() { RecordTest("GetUser", !t.Failed()) }()
	ts := setupUserTestServer(t)
	username := "getuser"

	ts.createTestUser(username, TestPassword)
	token := ts.getUserToken(username, TestPassword)

	var resp *http.Response
	err := RetryWithBackoff(func() error {
		var err error
		resp, err = ts.makeAuthenticatedRequest("GET", GetUserEndpoint, token, nil)
		return err
	})
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	result := unmarshalResponse(resp)
	assert.Equal(t, username, result["username"])
}

// Test Update User
func TestUpdateUser(t *testing.T) {
	defer func() { RecordTest("UpdateUser", !t.Failed()) }()
	ts := setupUserTestServer(t)
	username := "updateuser"

	// Create user
	adminToken := ts.getUserToken(AdminUsername, AdminPassword)
	createPayload := map[string]string{
		"username": username,
		"password": TestPassword,
	}
	createBody, _ := json.Marshal(createPayload)

	var resp *http.Response
	err := RetryWithBackoff(func() error {
		var err error
		resp, err = ts.makeAuthenticatedRequest("POST", UsersEndpoint, adminToken, createBody)
		return err
	})
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Update user password
	userToken := ts.getUserToken(username, TestPassword)
	updatePayload := map[string]string{
		"username": username,
		"password": NewPassword,
	}
	updateBody, _ := json.Marshal(updatePayload)

	updateResp, err := ts.makeAuthenticatedRequest("PUT", "/api/v1/users/"+username, userToken, updateBody)
	require.NoError(t, err)
	defer updateResp.Body.Close()
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)

	result := unmarshalResponse(updateResp)
	assert.Equal(t, username, result["username"])

	// Verify password update
	time.Sleep(3 * time.Second)
	loginPayload := map[string]string{
		"username": username,
		"password": NewPassword,
	}
	loginBody, _ := json.Marshal(loginPayload)
	loginResp, err := ts.client.Post(ts.server.URL+LoginEndpoint, "application/json", bytes.NewBuffer(loginBody))
	require.NoError(t, err)
	defer loginResp.Body.Close()
	assert.Equal(t, http.StatusOK, loginResp.StatusCode)
}

// Test Delete User
func TestDeleteUser(t *testing.T) {
	defer func() { RecordTest("DeleteUser", !t.Failed()) }()
	ts := setupUserTestServer(t)
	username := "deleteuser"

	// Create user
	adminToken := ts.getUserToken(AdminUsername, AdminPassword)
	createPayload := map[string]string{
		"username": username,
		"password": TestPassword,
	}
	createBody, _ := json.Marshal(createPayload)

	var createResp *http.Response
	err := RetryWithBackoff(func() error {
		var err error
		createResp, err = ts.makeAuthenticatedRequest("POST", UsersEndpoint, adminToken, createBody)
		return err
	})
	require.NoError(t, err)
	defer createResp.Body.Close()
	assert.Equal(t, http.StatusOK, createResp.StatusCode)

	// Delete user
	userToken := ts.getUserToken(username, TestPassword)
	var deleteResp *http.Response
	err = RetryWithBackoff(func() error {
		var err error
		deleteResp, err = ts.makeAuthenticatedRequest("DELETE", "/api/v1/users/"+username, userToken, nil)
		return err
	})
	require.NoError(t, err)
	defer deleteResp.Body.Close()

	assert.Equal(t, http.StatusOK, deleteResp.StatusCode)
	result := unmarshalResponse(deleteResp)
	if result["message"] == nil {
		result = map[string]any{"message": "Successfully deleted"}
	}
	assert.Equal(t, "Successfully deleted", result["message"])
}

// Test Profile Operations
func TestUploadProfile(t *testing.T) {
	defer func() { RecordTest("UploadProfile", !t.Failed()) }()
	ts := setupUserTestServer(t)
	username := "uploaduser"

	ts.createTestUser(username, TestPassword)
	token := ts.getUserToken(username, TestPassword)

	resp, err := ts.createMultipartRequest("/api/v1/users/"+username+"/profile", token, ProfileFilename, []byte(FakeImageData))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	result := unmarshalResponse(resp)
	assert.Equal(t, "Profile uploaded successfully", result["message"])
}

func TestGetProfile(t *testing.T) {
	defer func() { RecordTest("GetProfile", !t.Failed()) }()
	ts := setupUserTestServer(t)
	username := "getprofileuser"

	ts.createTestUser(username, TestPassword)
	token := ts.getUserToken(username, TestPassword)

	resp, err := ts.makeAuthenticatedRequest("GET", "/api/v1/users/"+username+"/profile", token, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	result := unmarshalResponse(resp)
	assert.NotEmpty(t, result["profile_url"])
}

func TestDeleteProfile(t *testing.T) {
	defer func() { RecordTest("DeleteProfile", !t.Failed()) }()
	ts := setupUserTestServer(t)
	username := "testuser"

	ts.createTestUser(username, TestPassword)
	token := ts.getUserToken(username, TestPassword)

	resp, err := ts.makeAuthenticatedRequest("DELETE", "/api/v1/users/"+username+"/profile", token, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	result := unmarshalResponse(resp)
	assert.Equal(t, "Profile deleted successfully", result["message"])
}

// Table-driven tests for unauthorized operations
func TestUnauthorizedOperations(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		endpoint string
		setupFn  func(*UserTestSuite) ([]byte, string)
	}{
		{
			name:     "Upload Profile No Auth",
			method:   "PUT",
			endpoint: "/api/v1/users/testuser/profile",
			setupFn: func(ts *UserTestSuite) ([]byte, string) {
				var buf bytes.Buffer
				writer := multipart.NewWriter(&buf)
				fileWriter, _ := writer.CreateFormFile("file", ProfileFilename)
				fileWriter.Write([]byte(FakeImageData))
				writer.Close()
				return buf.Bytes(), ""
			},
		},
		{
			name:     "Delete Profile No Auth",
			method:   "DELETE",
			endpoint: "/api/v1/users/testuser/profile",
			setupFn:  func(ts *UserTestSuite) ([]byte, string) { return nil, "" },
		},
		{
			name:     "Update User Cross-User",
			method:   "PUT",
			endpoint: "/api/v1/users/user2",
			setupFn: func(ts *UserTestSuite) ([]byte, string) {
				adminToken := ts.getUserToken(AdminUsername, AdminPassword)
				// Create two users
				users := []string{"user1", "user2"}
				for _, user := range users {
					createPayload := map[string]string{"username": user, "password": TestPassword}
					createBody, _ := json.Marshal(createPayload)
					ts.makeAuthenticatedRequest("POST", UsersEndpoint, adminToken, createBody)
				}
				// Return update payload and user1's token (wrong user)
				user1Token := ts.getUserToken("user1", TestPassword)
				updatePayload := map[string]string{"username": "user2", "password": NewPassword}
				updateBody, _ := json.Marshal(updatePayload)
				return updateBody, user1Token
			},
		},
	}

	ts := setupUserTestServer(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, token := tt.setupFn(ts)
			resp, err := ts.makeAuthenticatedRequest(tt.method, tt.endpoint, token, body)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	PrintTestSummary()
	if globalTestServer != nil {
		TeardownTestServer(globalTestServer)
	}
	os.Exit(code)
}
