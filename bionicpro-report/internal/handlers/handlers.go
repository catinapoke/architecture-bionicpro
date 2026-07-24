package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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

type ReportURLResponse struct {
	URL string `json:"url"`
}

type Verifier interface {
	Username(ctx context.Context, token string) (string, error)
}

type Store interface {
	ReportByUsername(ctx context.Context, username string) (Report, error)
}

type ObjectStore interface {
	Exists(ctx context.Context, key string) (bool, error)
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	CDNURL(key string) string
}

type Handler struct {
	verifier Verifier
	store    Store
	objects  ObjectStore
	now      func() time.Time
}

func New(verifier Verifier, store Store, objects ObjectStore) *Handler {
	return &Handler{
		verifier: verifier,
		store:    store,
		objects:  objects,
		now:      time.Now,
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

func reportObjectKey(username string, at time.Time) string {
	return username + "/" + at.UTC().Format("2006-01-02-15") + ".json"
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

	key := reportObjectKey(username, h.now())
	exists, err := h.objects.Exists(r.Context(), key)
	if err != nil {
		log.Printf("s3 exists check failed: %v", err)
		http.Error(w, "failed to check report storage", http.StatusInternalServerError)
		return
	}
	if exists {
		writeJSON(w, http.StatusOK, ReportURLResponse{URL: h.objects.CDNURL(key)})
		return
	}

	report, err := h.store.ReportByUsername(r.Context(), username)
	if errors.Is(err, ErrReportNotFound) {
		report = Report{
			Username:     username,
			ProthesisIDs: []string{},
		}
	} else if err != nil {
		log.Printf("report lookup failed: %v", err)
		http.Error(w, "failed to load report", http.StatusInternalServerError)
		return
	}

	payload, err := json.Marshal(report)
	if err != nil {
		log.Printf("report marshal failed: %v", err)
		http.Error(w, "failed to store report", http.StatusInternalServerError)
		return
	}

	if err := h.objects.Put(r.Context(), key, bytes.NewReader(payload), int64(len(payload)), "application/json"); err != nil {
		log.Printf("s3 put failed: %v", err)
		http.Error(w, "failed to store report", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, ReportURLResponse{URL: h.objects.CDNURL(key)})
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
