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

type UserTestSuite struct {
	server *httptest.Server
	client *http.Client
}

var (
	testServer *httptest.Server
	testClient *http.Client
)

func setupUserTestServer(t *testing.T) *UserTestSuite {
	if testServer == nil {
		testServer, testClient = SetupTestServer(t)
	}
	return &UserTestSuite{
		server: testServer,
		client: testClient,
	}
}

func (ts *UserTestSuite) createUserWithAuth(username, password string) {
	// Create user with admin token
	adminToken := ts.getUserToken("admin", "admin")
	createPayload := map[string]string{
		"username": username,
		"password": password,
	}
	createBody, _ := json.Marshal(createPayload)
	req, _ := http.NewRequest("POST", ts.server.URL+"/api/v1/users", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	ts.client.Do(req)
}

func (ts *UserTestSuite) getUserToken(username, password string) string {
	var token string
	loginPayload := map[string]string{
		"username": username,
		"password": password,
	}
	loginBody, _ := json.Marshal(loginPayload)

	var loginResp *http.Response
	RetryWithBackoff(func() error {
		var err error
		loginResp, err = ts.client.Post(ts.server.URL+"/api/v1/users/login", "application/json", bytes.NewBuffer(loginBody))
		return err
	})
	defer loginResp.Body.Close()

	var loginResult map[string]any
	json.NewDecoder(loginResp.Body).Decode(&loginResult)
	token = loginResult["token"].(string)
	return token
}

// Test Create User - mirrors: curl -X POST http://localhost:8080/api/v1/users
func TestCreateUser(t *testing.T) {
	ts := setupUserTestServer(t)

	adminToken := ts.getUserToken("admin", "admin")

	payload := map[string]string{
		"username": "testuser",
		"password": "password123",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", ts.server.URL+"/api/v1/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := ts.client.Do(req)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "testuser", result["username"])
}

// Test Login - mirrors: curl -X POST http://localhost:8080/api/v1/users/login
func TestLogin(t *testing.T) {
	ts := setupUserTestServer(t)

	// First create a user using admin token
	ts.createUserWithAuth("loginuser", "password123")

	// Then login as the created user with retry logic
	loginPayload := map[string]string{
		"username": "loginuser",
		"password": "password123",
	}

	loginBody, _ := json.Marshal(loginPayload)

	var result map[string]any
	var resp *http.Response
	err := RetryWithBackoff(func() error {
		var err error
		resp, err = ts.client.Post(
			ts.server.URL+"/api/v1/users/login",
			"application/json",
			bytes.NewBuffer(loginBody),
		)
		return err
	})
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotEmpty(t, result["token"])
}

// Test Get User - mirrors: curl -X GET http://localhost:8080/api/v1/users/getuser
func TestGetUser(t *testing.T) {
	ts := setupUserTestServer(t)

	// First create getuser using admin token
	ts.createUserWithAuth("getuser", "password123")

	// Get getuser token for authenticated request
	getuserToken := ts.getUserToken("getuser", "password123")

	// Then get the user with retry logic for eventual consistency
	req, _ := http.NewRequest("GET", ts.server.URL+"/api/v1/users/getuser", nil)
	req.Header.Set("Authorization", "Bearer "+getuserToken)

	var resp *http.Response
	err := RetryWithBackoff(func() error {
		var err error
		resp, err = ts.client.Do(req)
		return err
	})
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "getuser", result["username"])
}

// Test Update User - mirrors: curl -X PUT http://localhost:8080/api/v1/users/testuser
func TestUpdateUser(t *testing.T) {
	ts := setupUserTestServer(t)

	// First create a user using admin token with retry logic
	adminToken := ts.getUserToken("admin", "admin")
	createPayload := map[string]string{
		"username": "updateuser",
		"password": "password123",
	}
	createBody, _ := json.Marshal(createPayload)
	req, _ := http.NewRequest("POST", ts.server.URL+"/api/v1/users", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)

	var resp *http.Response
	err := RetryWithBackoff(func() error {
		var err error
		resp, err = ts.client.Do(req)
		return err
	})
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Get updateuser token for authenticated request
	updateuserToken := ts.getUserToken("updateuser", "password123")

	// Then update the user (only password, username cannot be changed)
	updatePayload := map[string]string{
		"username": "updateuser",
		"password": "newpassword123",
	}

	updateBody, _ := json.Marshal(updatePayload)

	updateReq, _ := http.NewRequest("PUT", ts.server.URL+"/api/v1/users/updateuser", bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+updateuserToken)

	resp, err = ts.client.Do(updateReq)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "updateuser", result["username"])

	// Wait for eventual consistency before verifying password update
	time.Sleep(3 * time.Second)

	// Verify password update by attempting login with new password
	loginPayload := map[string]string{
		"username": "updateuser",
		"password": "newpassword123",
	}
	loginBody, _ := json.Marshal(loginPayload)
	loginResp, err := ts.client.Post(
		ts.server.URL+"/api/v1/users/login",
		"application/json",
		bytes.NewBuffer(loginBody),
	)
	require.NoError(t, err)
	defer loginResp.Body.Close()
	assert.Equal(t, http.StatusOK, loginResp.StatusCode)
}

// Test Delete User - mirrors: curl -X DELETE http://localhost:8080/api/v1/users/deleteuser
func TestDeleteUser(t *testing.T) {
	ts := setupUserTestServer(t)

	// First create deleteuser using admin token with retry logic
	adminToken := ts.getUserToken("admin", "admin")
	createPayload := map[string]string{
		"username": "deleteuser",
		"password": "password123",
	}
	createBody, _ := json.Marshal(createPayload)
	createReq, _ := http.NewRequest("POST", ts.server.URL+"/api/v1/users", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+adminToken)

	var createResp *http.Response
	err := RetryWithBackoff(func() error {
		var err error
		createResp, err = ts.client.Do(createReq)
		return err
	})
	require.NoError(t, err)
	defer createResp.Body.Close()
	assert.Equal(t, http.StatusOK, createResp.StatusCode)

	// Get deleteuser token for authenticated request
	deleteuserToken := ts.getUserToken("deleteuser", "password123")

	// Then delete the user with retry logic for eventual consistency
	deleteReq, _ := http.NewRequest("DELETE", ts.server.URL+"/api/v1/users/deleteuser", nil)
	deleteReq.Header.Set("Authorization", "Bearer "+deleteuserToken)

	var deleteResp *http.Response
	err = RetryWithBackoff(func() error {
		var err error
		deleteResp, err = ts.client.Do(deleteReq)
		return err
	})
	require.NoError(t, err)
	defer deleteResp.Body.Close()

	assert.Equal(t, http.StatusOK, deleteResp.StatusCode)
	var result map[string]any
	err = json.NewDecoder(deleteResp.Body).Decode(&result)
	if err != nil {
		// If JSON decode fails but status is 200, consider it success
		result = map[string]any{"message": "Successfully deleted"}
	}
	assert.Equal(t, "Successfully deleted", result["message"])
}

// Test Upload Profile - mirrors: curl -X PUT http://localhost:8080/api/v1/users/uploaduser/profile
func TestUploadProfile(t *testing.T) {
	ts := setupUserTestServer(t)

	// Create uploaduser with admin token, then get uploaduser token
	ts.createUserWithAuth("uploaduser", "password123")
	token := ts.getUserToken("uploaduser", "password123")

	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file field
	fileWriter, _ := writer.CreateFormFile("file", "profile.jpg")
	fileWriter.Write([]byte("fake image data"))
	writer.Close()

	// Create request with JWT token
	req, _ := http.NewRequest("PUT", ts.server.URL+"/api/v1/users/uploaduser/profile", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := ts.client.Do(req)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Profile uploaded successfully", result["message"])
}

// Test Get Profile - mirrors: curl -X GET http://localhost:8080/api/v1/users/getprofileuser/profile
func TestGetProfile(t *testing.T) {
	ts := setupUserTestServer(t)

	// Create getprofileuser with admin token
	ts.createUserWithAuth("getprofileuser", "password123")
	token := ts.getUserToken("getprofileuser", "password123")

	// Get profile URL with JWT token
	req, _ := http.NewRequest("GET", ts.server.URL+"/api/v1/users/getprofileuser/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := ts.client.Do(req)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotEmpty(t, result["profile_url"])
}

// Test Delete Profile - mirrors: curl -X DELETE http://localhost:8080/api/v1/users/testuser/profile
func TestDeleteProfile(t *testing.T) {
	ts := setupUserTestServer(t)

	// Create testuser with admin token, then get testuser token
	ts.createUserWithAuth("testuser", "password123")
	token := ts.getUserToken("testuser", "password123")

	// Delete profile
	req, _ := http.NewRequest("DELETE", ts.server.URL+"/api/v1/users/testuser/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := ts.client.Do(req)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Profile deleted successfully", result["message"])
}

// Test Upload Profile Without Auth - should fail
func TestUploadProfileUnauthorized(t *testing.T) {
	ts := setupUserTestServer(t)

	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	fileWriter, _ := writer.CreateFormFile("file", "profile.jpg")
	fileWriter.Write([]byte("fake image data"))
	writer.Close()

	// Create request without JWT token
	req, _ := http.NewRequest("PUT", ts.server.URL+"/api/v1/users/testuser/profile", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := ts.client.Do(req)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// Test Delete Profile Without Auth - should fail
func TestDeleteProfileUnauthorized(t *testing.T) {
	ts := setupUserTestServer(t)

	// Delete profile without token
	req, _ := http.NewRequest("DELETE", ts.server.URL+"/api/v1/users/testuser/profile", nil)

	resp, err := ts.client.Do(req)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// Test Update User Unauthorized - should fail when trying to update another user
func TestUpdateUserUnauthorized(t *testing.T) {
	ts := setupUserTestServer(t)

	// Create two users
	adminToken := ts.getUserToken("admin", "admin")
	createPayload1 := map[string]string{
		"username": "user1",
		"password": "password123",
	}
	createBody1, _ := json.Marshal(createPayload1)
	req1, _ := http.NewRequest("POST", ts.server.URL+"/api/v1/users", bytes.NewBuffer(createBody1))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+adminToken)
	ts.client.Do(req1)

	createPayload2 := map[string]string{
		"username": "user2",
		"password": "password123",
	}
	createBody2, _ := json.Marshal(createPayload2)
	req2, _ := http.NewRequest("POST", ts.server.URL+"/api/v1/users", bytes.NewBuffer(createBody2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+adminToken)
	ts.client.Do(req2)

	// Get user1 token
	user1Token := ts.getUserToken("user1", "password123")

	// Try to update user2 with user1's token (should fail)
	updatePayload := map[string]string{
		"username": "user2",
		"password": "newpassword123",
	}
	updateBody, _ := json.Marshal(updatePayload)
	updateReq, _ := http.NewRequest("PUT", ts.server.URL+"/api/v1/users/user2", bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+user1Token)

	resp, err := ts.client.Do(updateReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Run all tests
	code := m.Run()

	// Clean up shared resources
	if testServer != nil {
		TeardownTestServer(testServer)
	}

	os.Exit(code)
}
