package main

import (
	"encoding/json"
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
	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		hitcount := asd.fileserverHits.Load()
		w.Write([]byte(fmt.Sprintf(`<html>
  										<body>
    										<h1>Welcome, Chirpy Admin</h1>
    										<p>Chirpy has been visited %d times!</p>
  										</body>
									</html>`, hitcount)))
	})
	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		asd.fileserverHits.Store(0)
		w.Write([]byte(fmt.Sprintf("Hits: %v", asd.fileserverHits.Load())))
	})
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		type jsondata struct {
			Body string `json:"body"`
		}
		type errordata struct {
			Error string `json:"error"`
		}
		type rspvalid struct {
			Valid bool `json:"valid"`
		}
		decoder := json.NewDecoder(r.Body)
		params := jsondata{}
		var check = 0
		err := decoder.Decode(&params)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			errResp := errordata{Error: "Something went wrong"}
			errjson, _ := json.Marshal(errResp)
			w.WriteHeader(500)
			w.Write(errjson)
			check = 1
		}
		if len(params.Body) > 140 {
			w.Header().Set("Content-Type", "application/json")
			errlen := errordata{Error: "Chirp is too long"}
			errlenjson, _ := json.Marshal(errlen)
			w.WriteHeader(400)
			w.Write(errlenjson)
			check = 1
		}
		if check == 0 {
			w.Header().Set("Content-Type", "application/json")
			corres := rspvalid{Valid: true}
			corresjson, _ := json.Marshal(corres)
			w.WriteHeader(200)
			w.Write(corresjson)
		}

	})
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	server.ListenAndServe()
}
