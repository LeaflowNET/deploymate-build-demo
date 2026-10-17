package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var version = "dev"
var commit = "unknown"

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	check := flag.Bool("healthcheck", false, "check the running HTTP service")
	flag.Parse()
	port := env("PORT", "8080")
	if *check {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("http://127.0.0.1:" + port + "/healthz")
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Fatalf("healthcheck: HTTP %d", resp.StatusCode)
		}
		return
	}
	started := time.Now().UTC()
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if err := json.NewEncoder(w).Encode(struct {
			Service   string    `json:"service"`
			Message   string    `json:"message"`
			Version   string    `json:"version"`
			Commit    string    `json:"commit"`
			Hostname  string    `json:"hostname"`
			StartedAt time.Time `json:"startedAt"`
		}{"deploymate-build-demo", env("DEMO_MESSAGE", "Hello from DeployMate!"), version, commit, hostname, started}); err != nil {
			log.Printf("write response: %v", err)
		}
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "ok")
	})
	server := &http.Server{Addr: ":" + port, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	log.Printf("listening on :%s version=%s commit=%s", port, version, commit)
	select {
	case err := <-done:
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatal(err)
		}
	}
}
