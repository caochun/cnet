package api

import (
	"net/http"
)

// handleResources handles resource information requests
func (s *Server) handleResources(w http.ResponseWriter, r *http.Request) {
	resources, err := s.resources.GetResources()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to get resources", err)
		return
	}

	s.writeJSON(w, http.StatusOK, resources)
}

// handleResourceUsage handles resource usage requests
func (s *Server) handleResourceUsage(w http.ResponseWriter, r *http.Request) {
	usage, err := s.resources.GetUsage()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to get resource usage", err)
		return
	}

	s.writeJSON(w, http.StatusOK, usage)
}
