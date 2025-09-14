package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"
	"github.com/zzenonn/go-zenon-api-aws/internal/config"
	"github.com/zzenonn/go-zenon-api-aws/internal/domain"
)

type UserService interface {
	GetUser(ctx context.Context, id string) (domain.User, error)
	UpdateUser(ctx context.Context, u domain.User) (domain.User, error)
	DeleteUser(ctx context.Context, id string) error
	CreateUser(ctx context.Context, u domain.User) (domain.User, error)
	Login(ctx context.Context, username string, password string) error
	UploadProfile(ctx context.Context, username string, key string, r io.Reader) error
	GeneratePresignedURL(ctx context.Context, username string, key string) (string, error)
	DeleteProfile(ctx context.Context, username string, key string) error
}

type Token struct {
	Token string `json:"token,omitempty"`
}

type UserHandler struct {
	Service    UserService
	JwtSignKey []byte
	Config     *config.Config
}

type PostUserRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

func convertPostUserRequestToUser(u PostUserRequest) domain.User {
	return domain.User{
		Username: &u.Username,
		Password: u.Password,
	}
}

func NewUserHandler(s UserService, cfg *config.Config) *UserHandler {
	h := &UserHandler{
		Service: s,
		Config:  cfg,
	}

	return h
}



func (h *UserHandler) PostUser(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received POST /api/v1/users request")

	var u PostUserRequest
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		log.Error("Error decoding request body: ", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	validate := validator.New()
	err := validate.Struct(u)
	if err != nil {
		log.Debug("Validation failed for user: ", u)
		http.Error(w, "Not a valid user", http.StatusBadRequest)
		return
	}

	convertedUser := convertPostUserRequestToUser(u)
	log.Debug(fmt.Sprintf("Converted user data: %#v", convertedUser))

	createdUser, err := h.Service.CreateUser(r.Context(), convertedUser)
	if err != nil {
		log.Error("Error creating user: ", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	log.Debug(fmt.Sprintf("User created successfully: %#v", createdUser))

	if err := json.NewEncoder(w).Encode(createdUser); err != nil {
		log.Fatal("Error encoding response: ", err)
	}
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received GET /api/v1/users/{username} request")

	username := chi.URLParam(r, "username")
	if username == "" {
		log.Error("No username provided in request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Debug(fmt.Sprintf("Fetching user with ID: %s", username))
	u, err := h.Service.GetUser(r.Context(), username)
	if err != nil {
		log.Error("Error fetching user: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Debug(fmt.Sprintf("User fetched successfully: %#v", u))

	if err := json.NewEncoder(w).Encode(u); err != nil {
		log.Fatal("Error encoding response: ", err)
	}
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received PUT /api/v1/users/{username} request")

	username := chi.URLParam(r, "username")
	if username == "" {
		log.Debug("No username provided in request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !ValidateUserAccess(w, r, username) {
		return
	}

	var req PostUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("Error decoding request body: ", err)
		return
	}

	// Validate that username in request body matches path parameter
	if req.Username != username {
		log.Error("Username in request body does not match path parameter")
		http.Error(w, "Username cannot be changed", http.StatusBadRequest)
		return
	}

	u := convertPostUserRequestToUser(req)

	log.Debug(fmt.Sprintf("Updating user with ID: %s", username))

	u, err := h.Service.UpdateUser(r.Context(), u)
	if err != nil {
		log.Error("Error updating user: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Debug(fmt.Sprintf("User updated successfully: %#v", u))

	if err := json.NewEncoder(w).Encode(u); err != nil {
		log.Fatal("Error encoding response: ", err)
	}
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received DELETE /api/v1/users/{username} request")

	username := chi.URLParam(r, "username")
	if username == "" {
		log.Debug("No username provided in request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Debug(fmt.Sprintf("Deleting user with ID: %s", username))

	err := h.Service.DeleteUser(r.Context(), username)
	if err != nil {
		log.Error("Error deleting user: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Debug("User deleted successfully")

	if err := json.NewEncoder(w).Encode(Response{Message: "Successfully deleted"}); err != nil {
		log.Fatal("Error encoding response: ", err)
	}
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received POST /api/v1/users/login request")

	var m map[string]string
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		log.Error("Error decoding request body: ", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	username := m["username"]
	password := m["password"]

	log.Debug(fmt.Sprintf("Attempting login for user: %s", username))

	err := h.Service.Login(r.Context(), username, password)
	if err != nil {
		log.Error("Login failed: ", err)
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	tokenString, err := generateJwtToken(username, h.Config.ECDSAPrivateKey)
	if err != nil {
		log.Error("Error generating JWT token: ", err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	log.Debug("JWT token generated successfully")

	token := Token{
		Token: tokenString,
	}

	if err := json.NewEncoder(w).Encode(token); err != nil {
		log.Error("Error encoding response: ", err)
	}
}

// PutProfile handles PUT requests to upload a user's profile image
func (h *UserHandler) PutProfile(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received PUT /api/v1/users/{username}/profile request")

	username := chi.URLParam(r, "username")
	if username == "" {
		log.Error("No username provided in request")
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	if !ValidateUserAccess(w, r, username) {
		return
	}

	// Parse multipart form with 10MB size limit
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Error("Failed to parse multipart form: ", err)
		http.Error(w, "Invalid multipart form", http.StatusBadRequest)
		return
	}

	// Retrieve and validate file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Error("Failed to read file: ", err)
		http.Error(w, "File upload is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get file extension
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".jpg" // default extension
	}
	key := "profile" + ext

	// Upload profile image using the service
	err = h.Service.UploadProfile(r.Context(), username, key, file)
	if err != nil {
		log.Error("Error uploading profile: ", err)
		http.Error(w, "Failed to upload profile", http.StatusInternalServerError)
		return
	}

	log.Debug(fmt.Sprintf("Profile uploaded successfully for user: %s", username))
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Response{Message: "Profile uploaded successfully"})
}

// GetProfile handles GET requests to retrieve a user's profile image URL
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received GET /api/v1/users/{username}/profile request")

	username := chi.URLParam(r, "username")
	if username == "" {
		log.Error("No username provided in request")
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Generate pre-signed URL for profile image
	profileURL, err := h.Service.GeneratePresignedURL(r.Context(), username, "profile.jpg")
	if err != nil {
		log.Error("Error generating profile URL: ", err)
		http.Error(w, "Failed to get profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := struct {
		ProfileURL string `json:"profile_url"`
	}{
		ProfileURL: profileURL,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("Error encoding response: ", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// DeleteProfile handles DELETE requests to remove a user's profile image
func (h *UserHandler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received DELETE /api/v1/users/{username}/profile request")

	username := chi.URLParam(r, "username")
	if username == "" {
		log.Error("No username provided in request")
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	if !ValidateUserAccess(w, r, username) {
		return
	}

	// Delete profile image using the service
	err := h.Service.DeleteProfile(r.Context(), username, "profile.jpg")
	if err != nil {
		log.Error("Error deleting profile: ", err)
		http.Error(w, "Failed to delete profile", http.StatusInternalServerError)
		return
	}

	log.Debug(fmt.Sprintf("Profile deleted successfully for user: %s", username))
	json.NewEncoder(w).Encode(Response{Message: "Profile deleted successfully"})
}

func (h *UserHandler) mapRoutes(router chi.Router) {
	router.Route("/api/v1/users", func(r chi.Router) {
		r.Post("/", JwtAuth(h.PostUser, h.Config.ECDSAPublicKey))
		r.Post("/login", h.Login)

		r.Route("/{username}", func(r chi.Router) {
			r.Get("/", JwtAuth(h.GetUser, h.Config.ECDSAPublicKey))
			r.Put("/", JwtAuth(h.UpdateUser, h.Config.ECDSAPublicKey))
			r.Delete("/", JwtAuth(h.DeleteUser, h.Config.ECDSAPublicKey))

			// Profile routes
			r.Route("/profile", func(r chi.Router) {
				r.Put("/", JwtAuth(h.PutProfile, h.Config.ECDSAPublicKey))
				r.Get("/", JwtAuth(h.GetProfile, h.Config.ECDSAPublicKey))
				r.Delete("/", JwtAuth(h.DeleteProfile, h.Config.ECDSAPublicKey))
			})
		})
	})
}
