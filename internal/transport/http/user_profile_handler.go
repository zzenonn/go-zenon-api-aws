package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
	"github.com/zzenonn/go-zenon-api-aws/internal/config"
)

type UserProfileService interface {
	UploadProfile(ctx context.Context, username string, key string, r io.Reader) error
	DownloadProfile(ctx context.Context, username string, key string) (io.ReadCloser, error)
	DeleteProfile(ctx context.Context, username string, key string) error
}

type UserProfileHandler struct {
	Service UserProfileService
	Config  *config.Config
}

func NewUserProfileHandler(s UserProfileService, cfg *config.Config) *UserProfileHandler {
	return &UserProfileHandler{
		Service: s,
		Config:  cfg,
	}
}

func (h *UserProfileHandler) PutProfile(w http.ResponseWriter, r *http.Request, sub string) {
	log.Debug("Received POST /api/v1/users/{username}/profile request")

	username := chi.URLParam(r, "username")
	if username == "" {
		log.Error("No username provided in request")
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	fileType := r.URL.Query().Get("type")
	if fileType == "" {
		fileType = "jpg" // Default file type
	}

	err := h.Service.UploadProfile(r.Context(), username, fileType, r.Body)
	if err != nil {
		log.Error("Error uploading profile: ", err)
		http.Error(w, "Failed to upload profile", http.StatusInternalServerError)
		return
	}

	log.Debug(fmt.Sprintf("Profile uploaded successfully for user: %s", username))
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Response{Message: "Profile uploaded successfully"})
}

func (h *UserProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	log.Debug("Received GET /api/v1/users/{username}/profile request")

	username := chi.URLParam(r, "username")
	if username == "" {
		log.Error("No username provided in request")
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	fileType := r.URL.Query().Get("type")
	if fileType == "" {
		fileType = "jpg" // Default file type
	}

	file, err := h.Service.DownloadProfile(r.Context(), username, fileType)
	if err != nil {
		log.Error("Error downloading profile: ", err)
		http.Error(w, "Failed to download profile", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	log.Debug(fmt.Sprintf("Profile downloaded successfully for user: %s", username))

	// Set appropriate content type based on file type
	contentType := "image/jpeg"
	if fileType == "png" {
		contentType = "image/png"
	}
	w.Header().Set("Content-Type", contentType)

	io.Copy(w, file)
}

func (h *UserProfileHandler) DeleteProfile(w http.ResponseWriter, r *http.Request, sub string) {
	log.Debug("Received DELETE /api/v1/users/{username}/profile request")

	username := chi.URLParam(r, "username")
	if username == "" {
		log.Error("No username provided in request")
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	fileType := r.URL.Query().Get("type")
	if fileType == "" {
		fileType = "jpg" // Default file type
	}

	err := h.Service.DeleteProfile(r.Context(), username, fileType)
	if err != nil {
		log.Error("Error deleting profile: ", err)
		http.Error(w, "Failed to delete profile", http.StatusInternalServerError)
		return
	}

	log.Debug(fmt.Sprintf("Profile deleted successfully for user: %s", username))
	json.NewEncoder(w).Encode(Response{Message: "Profile deleted successfully"})
}

func (h *UserProfileHandler) mapRoutes(router chi.Router) {
	router.Route("/api/v1/users/{username}/profile", func(r chi.Router) {
		r.Post("/", JwtAuth(h.PutProfile, h.Config.ECDSAPublicKey))
		r.Get("/", h.GetProfile)
		r.Delete("/", JwtAuth(h.DeleteProfile, h.Config.ECDSAPublicKey))
	})
}
