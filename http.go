package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}
func apiError(w http.ResponseWriter, status int, code string) {
	jsonResponse(w, status, struct {
		Error string `json:"error"`
	}{code})
}
func storeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errNotFound):
		apiError(w, 404, "not_found")
	case errors.Is(err, errFull):
		apiError(w, 409, "task_limit")
	default:
		log.Printf("task storage: %v", err)
		apiError(w, 500, "storage_error")
	}
}
func validStatus(s string) bool   { return s == "todo" || s == "doing" || s == "done" }
func validPriority(s string) bool { return s == "low" || s == "medium" || s == "high" }
func readInput(w http.ResponseWriter, r *http.Request) (TaskInput, bool) {
	var input TaskInput
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		apiError(w, 400, "invalid_input")
		return input, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		apiError(w, 400, "invalid_input")
		return input, false
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if utf8.RuneCountInString(input.Title) < 1 || utf8.RuneCountInString(input.Title) > 200 || utf8.RuneCountInString(input.Description) > 2000 || !validStatus(input.Status) || !validPriority(input.Priority) {
		apiError(w, 400, "invalid_input")
		return input, false
	}
	return input, true
}
func newHandler(store *Store, static http.Handler) http.Handler {
	mux := http.NewServeMux()
	started := time.Now().UTC()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /api/info", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, struct {
			Name      string    `json:"name"`
			Version   string    `json:"version"`
			Commit    string    `json:"commit"`
			Message   string    `json:"message"`
			StartedAt time.Time `json:"startedAt"`
		}{"Taskmate", version, commit, env("DEMO_MESSAGE", "Your next idea starts here."), started})
	})
	mux.HandleFunc("GET /api/tasks", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		status, priority := query.Get("status"), query.Get("priority")
		if status != "" && !validStatus(status) || priority != "" && !validPriority(priority) || len(query.Get("q")) > 1000 {
			apiError(w, 400, "invalid_query")
			return
		}
		limit, offset := 50, 0
		var err error
		if query.Has("limit") {
			limit, err = strconv.Atoi(query.Get("limit"))
			if err != nil || limit < 1 || limit > 100 {
				apiError(w, 400, "invalid_query")
				return
			}
		}
		if query.Has("offset") {
			offset, err = strconv.Atoi(query.Get("offset"))
			if err != nil || offset < 0 || offset > maxTasks {
				apiError(w, 400, "invalid_query")
				return
			}
		}
		jsonResponse(w, 200, store.list(query.Get("q"), status, priority, offset, limit))
	})
	mux.HandleFunc("POST /api/tasks", func(w http.ResponseWriter, r *http.Request) {
		input, ok := readInput(w, r)
		if !ok {
			return
		}
		task, err := store.put("", input)
		if err != nil {
			storeError(w, err)
			return
		}
		jsonResponse(w, 201, task)
	})
	mux.HandleFunc("PUT /api/tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		input, ok := readInput(w, r)
		if !ok {
			return
		}
		task, err := store.put(r.PathValue("id"), input)
		if err != nil {
			storeError(w, err)
			return
		}
		jsonResponse(w, 200, task)
	})
	mux.HandleFunc("DELETE /api/tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := store.delete(r.PathValue("id")); err != nil {
			storeError(w, err)
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { apiError(w, 404, "not_found") })
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		static.ServeHTTP(w, r)
	}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		if r.Header.Get("Origin") != "" && r.Method != "GET" && r.Method != "HEAD" {
			origin := r.Header.Get("Origin")
			if origin != "http://"+r.Host && origin != "https://"+r.Host {
				apiError(w, 403, "invalid_origin")
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}
