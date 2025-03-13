package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/mgenc2077/bootdev-chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
}
type errordata struct {
	Error string `json:"error"`
}
type emailquery struct {
	Email string `json:"email"`
}
type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}
type chirpsInput struct {
	Body   string    `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}
type chirpsOutput struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

var apiconfig *apiConfig

func returnwitherror(w http.ResponseWriter, code int, msg string) int {
	w.Header().Set("Content-Type", "application/json")
	check := 1
	errResp := errordata{Error: msg}
	errjson, _ := json.Marshal(errResp)
	w.WriteHeader(code)
	w.Write(errjson)
	return check
}

func createChirp(w http.ResponseWriter, code int, bodydata chirpsInput, r *http.Request) {
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
	chirp, err := apiconfig.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{Body: rspstring, UserID: bodydata.UserID})
	if err != nil {
		returnwitherror(w, 500, "Could not create Chirp")
	}
	chirpresp := chirpsOutput{ID: chirp.ID, CreatedAt: chirp.CreatedAt, UpdatedAt: chirp.UpdatedAt, Body: chirp.Body, UserID: chirp.UserID}
	rspjson, err := json.Marshal(chirpresp)
	if err != nil {
		returnwitherror(w, 500, "Could not marshall chirpresp")
	}
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
	apiconfig = &apiConfig{dbQueries: database.New(db), platform: os.Getenv("PLATFORM")}
	mux.Handle("/app/", apiconfig.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.Handle("/assets/", http.FileServer(http.Dir(".")))
	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		hitcount := apiconfig.fileserverHits.Load()
		w.Write([]byte(fmt.Sprintf(`<html>
  										<body>
    										<h1>Welcome, Chirpy Admin</h1>
    										<p>Chirpy has been visited %d times!</p>
  										</body>
									</html>`, hitcount)))
	})
	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if apiconfig.platform == "dev" {
			apiconfig.dbQueries.ResetTable(r.Context())
			w.WriteHeader(http.StatusOK)
			apiconfig.fileserverHits.Store(0)
			w.Write([]byte(fmt.Sprintf("Hits: %v", apiconfig.fileserverHits.Load())))
		} else {
			w.WriteHeader(403)
		}
	})
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		params := chirpsInput{}
		var check = 0
		err := decoder.Decode(&params)
		if err != nil {
			check = returnwitherror(w, 500, "Something went wrong")
		}
		if len(params.Body) > 140 {
			check = returnwitherror(w, 400, "Chirp is too long")
		}
		if check == 0 {
			createChirp(w, 201, params, r)
		}
	})
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		params1 := emailquery{}
		err := decoder.Decode(&params1)
		if err != nil {
			_ = returnwitherror(w, 500, "Something went wrong")
		}
		user, err := apiconfig.dbQueries.CreateUser(r.Context(), params1.Email)
		if err != nil {
			returnwitherror(w, 500, "Could not create User")
		}
		userstruct := User{ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email}
		userjson, err := json.Marshal(userstruct)
		if err != nil {
			returnwitherror(w, 500, "Could not marshall userstruct")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write(userjson)
	})
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	server.ListenAndServe()
}
