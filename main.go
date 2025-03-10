package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiconfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiconfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()
	asd := &apiconfig{}
	mux.Handle("/app/", asd.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.Handle("/assets/", http.FileServer(http.Dir(".")))
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		hitcount := asd.fileserverHits.Load()
		w.Write([]byte(fmt.Sprintf("Hits: %v", hitcount)))
	})
	mux.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		asd.fileserverHits.Store(0)
		w.Write([]byte(fmt.Sprintf("Hits: %v", asd.fileserverHits.Load())))
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	server.ListenAndServe()
}
