package http

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

type Response struct {
	Message string `json:"message"`
}

// ValidateUserAccess checks if the JWT token's sub claim matches the username
func ValidateUserAccess(w http.ResponseWriter, r *http.Request, username string) bool {
	subVal := r.Context().Value(subjectContextKey)
	sub, ok := subVal.(string)
	if !ok || sub != username {
		log.Error("Token sub does not match username or sub is missing. Sub value: ", sub)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}
