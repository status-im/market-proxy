package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

// TokenListHandler handles requests for token lists by platform
func (s *Server) TokenListHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	platform := vars["platform"]

	if platform == "" {
		http.Error(w, "Platform parameter is required", http.StatusBadRequest)
		return
	}

	tokenListResponse := s.tokenListService.GetTokenList(platform)
	if tokenListResponse.Error != nil {
		http.Error(w, tokenListResponse.Error.Error(), http.StatusBadRequest)
		return
	}

	s.sendJSONResponse(w, tokenListResponse.TokenList)
}
