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
	// Marshal the data to calculate content length and ETag
	responseBytes, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}

	// Calculate ETag (MD5 hash of the response)
	hash := md5.Sum(responseBytes)
	etag := hex.EncodeToString(hash[:])

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(responseBytes)))
	w.Header().Set("ETag", "\""+etag+"\"")

	// Write the response
	if _, err := w.Write(responseBytes); err != nil {
		log.Printf("Error writing response: %v", err)
		return
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
