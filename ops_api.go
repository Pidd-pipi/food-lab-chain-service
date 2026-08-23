package main

import (
	"encoding/json"
	"log"
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
			opsJSON(w, 405, map[string]string{"code": "method_not_allowed", "error": "method not allowed"})
			return
		}
		if r.Method == http.MethodPost {
			opsCreateRecord(svc, w, r)
			return
		}
		query := opsQueryFromRequest(r)
		page, err := svc.Search(r.Context(), query)
		if err != nil {
			opsWriteError(w, r, err)
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
		opsJSON(w, 400, map[string]string{"code": "invalid", "error": "body must be valid JSON"})
		return
	}
	created, err := svc.Create(r.Context(), record)
	if err != nil {
		opsWriteError(w, r, err)
		return
	}
	opsJSON(w, 201, created)
}

func opsRecordByID(svc *OpsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := opsPathID(r.URL.Path, "/records/")
		if id == "" {
			opsJSON(w, 404, map[string]string{"code": "not_found", "error": "record id is required"})
			return
		}
		if r.Method == http.MethodPost && strings.HasSuffix(id, "/transition") {
			opsTransitionRecord(svc, w, r, strings.TrimSuffix(id, "/transition"))
			return
		}
		if r.Method != http.MethodGet {
			opsJSON(w, 405, map[string]string{"code": "method_not_allowed", "error": "method not allowed"})
			return
		}
		record, err := svc.Get(r.Context(), id)
		if err != nil {
			opsWriteError(w, r, err)
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
		opsJSON(w, 400, map[string]string{"code": "invalid", "error": "body must be valid JSON"})
		return
	}
	record, err := svc.Transition(r.Context(), id, body.Expected, body.Target, opsActorFromRequest(r))
	if err != nil {
		opsWriteError(w, r, err)
		return
	}
	opsJSON(w, 200, record)
}

func opsSnapshot(svc *OpsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			opsJSON(w, 405, map[string]string{"code": "method_not_allowed", "error": "method not allowed"})
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

// opsStatusForError maps a domain error to an HTTP status code by way of its
// stable category. Every known category resolves to a specific status so the
// frontend can distinguish "not found", "bad request", and "conflict" from a
// genuine server fault.
func opsStatusForError(err error) int {
	switch opsCode(err) {
	case "not_found":
		return 404
	case "invalid", "policy":
		return 400
	case "conflict", "transition":
		return 409
	default:
		return 500
	}
}

// opsErrorMessage returns the client-facing message for an error. Known
// categories surface a stable, human-readable hint so the response body does
// not leak internal operation names; unrecognized errors are reported as
// "internal" while the original error is logged server-side for diagnosis.
func opsErrorMessage(err error) string {
	switch opsCode(err) {
	case "not_found":
		return "record not found"
	case "invalid":
		return "request is invalid"
	case "policy":
		return "request rejected by policy"
	case "conflict":
		return "record conflicts with existing data"
	case "transition":
		return "status transition is not allowed"
	default:
		return "internal error"
	}
}

// opsWriteError classifies a service error and writes a structured error
// response. The body carries a stable "code" the frontend can branch on and a
// human-readable "error" message that never leaks internal operation names.
// The original error is logged with the request id so genuine faults remain
// diagnosable server-side instead of being flattened into an opaque 500.
func opsWriteError(w http.ResponseWriter, r *http.Request, err error) {
	code := opsCode(err)
	status := opsStatusForError(err)
	if status == 500 {
		log.Printf("ops error request_id=%s method=%s path=%s code=%s err=%v",
			r.Header.Get("X-Request-ID"), r.Method, r.URL.Path, code, err)
	}
	opsJSON(w, status, map[string]string{"code": code, "error": opsErrorMessage(err)})
}
