package api

import (
	"encoding/json"
	"errors"
	"food-lab-chain-service/domain"
	"food-lab-chain-service/store"
	"food-lab-chain-service/validation"
	"net/http"
	"strings"
)

func listSpecimens(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, 405, "method not allowed")
			return
		}
		writeJSON(w, 200, map[string]any{"specimens": s.List()})
	}
}
func handoffSpecimen(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/handoff") {
			writeError(w, 404, "endpoint not found")
			return
		}
		input, err := validation.DecodeHandoff(r)
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}
		updated, err := s.Handoff(specimenID(r.URL.Path), input.To)
		if err != nil {
			status := 409
			if errors.Is(err, domain.ErrSpecimenNotFound) {
				status = 404
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, 200, map[string]string{"status": "handoff-recorded", "specimenID": specimenID(r.URL.Path), "to": input.To, "chainState": updated.ChainState})
	}
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
