package integration

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type UserTestSuite struct {
	server *httptest.Server
	client *http.Client
}

func setupUserTestServer(t *testing.T) *UserTestSuite {
	server, client := SetupTestServer(t)
	return &UserTestSuite{
		server: server,
		client: client,
	}
}

func (ts *UserTestSuite) teardown(t *testing.T) {
	TeardownTestServer(ts.server, t)
}

func (ts *UserTestSuite) createUserAndGetToken(username, password string) string {
	// Create user
	createPayload := map[string]string{
		"username": username,
		"password": password,
	}
	createBody, _ := json.Marshal(createPayload)
	ts.client.Post(ts.server.URL+"/api/v1/users", "application/json", bytes.NewBuffer(createBody))

	// Login to get token
	loginResp, _ := ts.client.Post(ts.server.URL+"/api/v1/users/login", "application/json", bytes.NewBuffer(createBody))
	var loginResult map[string]interface{}
	json.NewDecoder(loginResp.Body).Decode(&loginResult)
	token := loginResult["token"].(string)
	loginResp.Body.Close()

	return token
}

// Test Create User - mirrors: curl -X POST http://localhost:8080/api/v1/users
func TestCreateUser(t *testing.T) {
	ts := setupUserTestServer(t)
	defer ts.teardown(t)

	payload := map[string]string{
		"username": "new-user",
		"password": "password123",
	}

	body, _ := json.Marshal(payload)

	resp, err := ts.client.Post(
		ts.server.URL+"/api/v1/users",
		"application/json",
		bytes.NewBuffer(body),
	)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "new-user", result["username"])
}

// Test Login - mirrors: curl -X POST http://localhost:8080/api/v1/users/login
func TestLogin(t *testing.T) {
	ts := setupUserTestServer(t)
	defer ts.teardown(t)

	// First create a user
	createPayload := map[string]string{
		"username": "testuser",
		"password": "password123",
	}
	createBody, _ := json.Marshal(createPayload)
	ts.client.Post(ts.server.URL+"/api/v1/users", "application/json", bytes.NewBuffer(createBody))

	// Then login
	loginPayload := map[string]string{
		"username": "testuser",
		"password": "password123",
	}

	loginBody, _ := json.Marshal(loginPayload)

	resp, err := ts.client.Post(
		ts.server.URL+"/api/v1/users/login",
		"application/json",
		bytes.NewBuffer(loginBody),
	)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotEmpty(t, result["token"])
}

// Test Get User - mirrors: curl -X GET http://localhost:8080/api/v1/users/testuser
func TestGetUser(t *testing.T) {
	ts := setupUserTestServer(t)
	defer ts.teardown(t)

	// First create a user
	createPayload := map[string]string{
		"username": "testuser",
		"password": "password123",
	}
	createBody, _ := json.Marshal(createPayload)
	ts.client.Post(ts.server.URL+"/api/v1/users", "application/json", bytes.NewBuffer(createBody))

	// Then get the user
	resp, err := ts.client.Get(ts.server.URL + "/api/v1/users/testuser")

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "testuser", result["username"])
}

// Test Update User - mirrors: curl -X PUT http://localhost:8080/api/v1/users/testuser
func TestUpdateUser(t *testing.T) {
	ts := setupUserTestServer(t)
	defer ts.teardown(t)

	// First create a user
	createPayload := map[string]string{
		"username": "testuser",
		"password": "password123",
	}
	createBody, _ := json.Marshal(createPayload)
	ts.client.Post(ts.server.URL+"/api/v1/users", "application/json", bytes.NewBuffer(createBody))

	// Then update the user
	updatePayload := map[string]string{
		"username": "updateduser",
		"password": "newpassword123",
	}

	updateBody, _ := json.Marshal(updatePayload)

	req, _ := http.NewRequest("PUT", ts.server.URL+"/api/v1/users/testuser", bytes.NewBuffer(updateBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.client.Do(req)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "updateduser", result["username"])
}

// Test Delete User - mirrors: curl -X DELETE http://localhost:8080/api/v1/users/testuser
func TestDeleteUser(t *testing.T) {
	ts := setupUserTestServer(t)
	defer ts.teardown(t)

	// First create a user
	createPayload := map[string]string{
		"username": "testuser",
		"password": "password123",
	}
	createBody, _ := json.Marshal(createPayload)
	ts.client.Post(ts.server.URL+"/api/v1/users", "application/json", bytes.NewBuffer(createBody))

	// Then delete the user
	req, _ := http.NewRequest("DELETE", ts.server.URL+"/api/v1/users/testuser", nil)

	resp, err := ts.client.Do(req)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Successfully deleted", result["message"])
}

// Test Upload Profile - mirrors: curl -X PUT http://localhost:8080/api/v1/users/new-user/profile
func TestUploadProfile(t *testing.T) {
	ts := setupUserTestServer(t)
	defer ts.teardown(t)

	token := ts.createUserAndGetToken("new-user", "password123")

	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file field
	fileWriter, _ := writer.CreateFormFile("file", "profile.jpg")
	fileWriter.Write([]byte("fake image data"))
	writer.Close()

	// Create request with JWT token
	req, _ := http.NewRequest("PUT", ts.server.URL+"/api/v1/users/new-user/profile", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := ts.client.Do(req)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Profile uploaded successfully", result["message"])
}

// Test Get Profile - mirrors: curl -X GET http://localhost:8080/api/v1/users/new-user/profile
func TestGetProfile(t *testing.T) {
	ts := setupUserTestServer(t)
	defer ts.teardown(t)

	// Create user first
	ts.createUserAndGetToken("new-user", "password123")

	// Get profile URL
	resp, err := ts.client.Get(ts.server.URL + "/api/v1/users/new-user/profile")

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotEmpty(t, result["profile_url"])
}

// Test Delete Profile - mirrors: curl -X DELETE http://localhost:8080/api/v1/users/new-user/profile
func TestDeleteProfile(t *testing.T) {
	ts := setupUserTestServer(t)
	defer ts.teardown(t)

	token := ts.createUserAndGetToken("new-user", "password123")

	// Delete profile
	req, _ := http.NewRequest("DELETE", ts.server.URL+"/api/v1/users/new-user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := ts.client.Do(req)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Profile deleted successfully", result["message"])
}

// Test Upload Profile Without Auth - should fail
func TestUploadProfileUnauthorized(t *testing.T) {
	ts := setupUserTestServer(t)
	defer ts.teardown(t)

	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	fileWriter, _ := writer.CreateFormFile("file", "profile.jpg")
	fileWriter.Write([]byte("fake image data"))
	writer.Close()

	// Create request without JWT token
	req, _ := http.NewRequest("PUT", ts.server.URL+"/api/v1/users/new-user/profile", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := ts.client.Do(req)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// Test Delete Profile Without Auth - should fail
func TestDeleteProfileUnauthorized(t *testing.T) {
	ts := setupUserTestServer(t)
	defer ts.teardown(t)

	// Delete profile without token
	req, _ := http.NewRequest("DELETE", ts.server.URL+"/api/v1/users/new-user/profile", nil)

	resp, err := ts.client.Do(req)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
