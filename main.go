package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/mgenc2077/bootdev-chirpy/internal/database"
)

type apiconfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
}
type errordata struct {
	Error string `json:"error"`
}
type jsondata struct {
	Body string `json:"body"`
}
type jsonresponse struct {
	Cleaned_body string `json:"cleaned_body"`
}

func (cfg *apiconfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func returnwitherror(w http.ResponseWriter, code int, msg string) int {
	w.Header().Set("Content-Type", "application/json")
	check := 1
	errResp := errordata{Error: msg}
	errjson, _ := json.Marshal(errResp)
	w.WriteHeader(code)
	w.Write(errjson)
	return check
}

func returnwithvalues(w http.ResponseWriter, code int, bodydata jsondata) {
	w.Header().Set("Content-Type", "application/json")
	arr := strings.Split(bodydata.Body, " ")
	var arres []string
	for _, v := range arr {
		val1 := strings.ToLower(v)
		if (val1 == "kerfuffle") || (val1 == "sharbert") || (val1 == "fornax") {
			arres = append(arres, "****")
			continue
		}
		arres = append(arres, v)
	}
	rspstring := strings.Join(arres, " ")
	response := jsonresponse{Cleaned_body: rspstring}
	rspjson, _ := json.Marshal(response)
	w.WriteHeader(code)
	w.Write(rspjson)
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return
	}
	mux := http.NewServeMux()
	asd := &apiconfig{dbQueries: database.New(db)}
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
		decoder := json.NewDecoder(r.Body)
		params := jsondata{}
		var check = 0
		err := decoder.Decode(&params)
		if err != nil {
			check = returnwitherror(w, 500, "Something went wrong")
		}
		if len(params.Body) > 140 {
			check = returnwitherror(w, 400, "Chirp is too long")
		}
		if check == 0 {
			returnwithvalues(w, 200, params)
		}
	})
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	server.ListenAndServe()
}
