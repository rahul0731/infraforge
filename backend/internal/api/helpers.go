package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// JSON writes a JSON response.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// Error writes a JSON error response.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}

// ParseUUID extracts and parses a UUID from a path segment.
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// GetTeamID extracts the team ID from the X-Team-ID header.
func GetTeamID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(r.Header.Get("X-Team-ID"))
}

// GetActor extracts the actor from the X-Actor header.
func GetActor(r *http.Request) string {
	actor := r.Header.Get("X-Actor")
	if actor == "" {
		return "system"
	}
	return actor
}
