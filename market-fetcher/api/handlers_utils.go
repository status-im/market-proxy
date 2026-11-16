package api

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// setCacheStatusHeader sets the Cache-Status header based on cache status
func (s *Server) setCacheStatusHeader(w http.ResponseWriter, cacheStatus string) {
	if cacheStatus != "" {
		w.Header().Set("Cache-Status", cacheStatus)
	}
}

// sendJSONResponse is a common wrapper for JSON responses that sets Content-Type,
// Content-Length and ETag headers
func (s *Server) sendJSONResponse(w http.ResponseWriter, data interface{}) {
	responseBytes, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
	s.sendJSONBytes(w, responseBytes)
}

// sendJSONBytes writes a body that is already JSON with the same headers as
// sendJSONResponse. The ETag is the MD5 of the body, so nginx can answer a
// matching If-None-Match with 304.
func (s *Server) sendJSONBytes(w http.ResponseWriter, body []byte) {
	hash := md5.Sum(body)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.Header().Set("ETag", "\""+hex.EncodeToString(hash[:])+"\"")

	if _, err := w.Write(body); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// sendJSONError writes a JSON error body with the given HTTP status
func (s *Server) sendJSONError(w http.ResponseWriter, status int, message string) {
	responseBytes, err := json.Marshal(map[string]string{"error": message})
	if err != nil {
		http.Error(w, message, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(responseBytes)))
	w.WriteHeader(status)

	if _, err := w.Write(responseBytes); err != nil {
		log.Printf("Error writing error response: %v", err)
	}
}

// Stop gracefully shuts down the server
func (s *Server) Stop() {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
	}
}

func getParamLowercase(r *http.Request, key string) string {
	if r == nil {
		return ""
	}
	value := r.URL.Query().Get(key)
	if value != "" {
		return strings.ToLower(value)
	}
	return ""
}

func splitParamLowercase(param string) []string {
	if param == "" {
		return []string{}
	}

	parts := strings.Split(param, ",")
	result := []string{}
	for _, part := range parts {
		trimmed := strings.ToLower(strings.TrimSpace(part))
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
