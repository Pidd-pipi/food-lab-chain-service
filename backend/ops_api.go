package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// opsRouter exposes the operations layer as an HTTP API under /api/ops.
func opsRouter(svc *OpsService) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/records", opsListRecords(svc))
	mux.HandleFunc("/records/", opsRecordByID(svc))
	mux.HandleFunc("/snapshot", opsSnapshot(svc))
	return mux
}

func opsListRecords(svc *OpsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !opsAllowed(r.Method, http.MethodGet, http.MethodPost) {
			opsJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		if r.Method == http.MethodPost {
			opsCreateRecord(svc, w, r)
			return
		}
		query := opsQueryFromRequest(r)
		page, err := svc.Search(r.Context(), query)
		if err != nil {
			opsJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		opsJSON(w, 200, map[string]any{
			"items":    page.Items,
			"page":     page.Page,
			"pageSize": page.PageSize,
			"total":    page.Total,
			"hasNext":  page.HasNext,
		})
	}
}

func opsCreateRecord(svc *OpsService, w http.ResponseWriter, r *http.Request) {
	var record OpsRecord
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		opsJSON(w, 400, map[string]string{"error": "body must be valid JSON"})
		return
	}
	created, err := svc.Create(r.Context(), record)
	if err != nil {
		opsJSON(w, opsStatusForError(err), map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, 201, created)
}

func opsRecordByID(svc *OpsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := opsPathID(r.URL.Path, "/records/")
		if id == "" {
			opsJSON(w, 404, map[string]string{"error": "record id is required"})
			return
		}
		if r.Method == http.MethodPost && strings.HasSuffix(id, "/transition") {
			opsTransitionRecord(svc, w, r, strings.TrimSuffix(id, "/transition"))
			return
		}
		if r.Method != http.MethodGet {
			opsJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		record, err := svc.Get(r.Context(), id)
		if err != nil {
			opsJSON(w, opsStatusForError(err), map[string]string{"error": err.Error()})
			return
		}
		opsJSON(w, 200, record)
	}
}

func opsTransitionRecord(svc *OpsService, w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Target   OpsStatus `json:"target"`
		Expected int       `json:"expected"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		opsJSON(w, 400, map[string]string{"error": "body must be valid JSON"})
		return
	}
	record, err := svc.Transition(r.Context(), id, body.Expected, body.Target, opsActorFromRequest(r))
	if err != nil {
		opsJSON(w, opsStatusForError(err), map[string]string{"error": err.Error()})
		return
	}
	opsJSON(w, 200, record)
}

func opsSnapshot(svc *OpsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			opsJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		opsJSON(w, 200, svc.Snapshot())
	}
}

func opsQueryFromRequest(r *http.Request) OpsQuery {
	q := OpsQuery{
		Subject:  r.URL.Query().Get("subject"),
		Owner:    r.URL.Query().Get("owner"),
		Status:   OpsStatus(r.URL.Query().Get("status")),
		Priority: OpsPriority(r.URL.Query().Get("priority")),
	}
	if page, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil {
		q.Page = page
	}
	if size, err := strconv.Atoi(r.URL.Query().Get("pageSize")); err == nil {
		q.PageSize = size
	}
	return q
}

// opsStatusForError maps a domain error to an HTTP status code. The mapping
// relies on the error chain being intact so errors.Is can recognize the
// sentinel errors.
func opsStatusForError(err error) int {
	switch opsCode(err) {
	case "not_found":
		return 404
	case "invalid":
		return 400
	case "transition":
		return 409
	case "policy":
		return 403
	case "conflict":
		return 409
	default:
		return 500
	}
}
