package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

var ErrReportNotFound = errors.New("report not found")

type Report struct {
	Username     string     `json:"username"`
	Name         string     `json:"name"`
	CreatedAt    *time.Time `json:"created_at"`
	ProthesisIDs []string   `json:"prothesis_ids"`
	SignalsCount int64      `json:"signals_count"`
}

type Verifier interface {
	Username(ctx context.Context, token string) (string, error)
}

type Store interface {
	ReportByUsername(ctx context.Context, username string) (Report, error)
}

type Handler struct {
	verifier Verifier
	store    Store
}

func New(verifier Verifier, store Store) *Handler {
	return &Handler{
		verifier: verifier,
		store:    store,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /reports", h.reports)
	return mux
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) reports(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	username, err := h.verifier.Username(r.Context(), token)
	if err != nil {
		log.Printf("access token verification failed: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	report, err := h.store.ReportByUsername(r.Context(), username)
	if errors.Is(err, ErrReportNotFound) {
		writeJSON(w, http.StatusOK, Report{
			Username:     username,
			ProthesisIDs: []string{},
		})
		return
	}
	if err != nil {
		log.Printf("report lookup failed: %v", err)
		http.Error(w, "failed to load report", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func bearerToken(header string) (string, bool) {
	prefix := "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	return token, token != ""
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
