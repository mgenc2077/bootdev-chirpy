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
	"github.com/mgenc2077/bootdev-chirpy/internal/auth"
	"github.com/mgenc2077/bootdev-chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
	jwt_Secret     string
}
type errordata struct {
	Error string `json:"error"`
}
type emailquery struct {
	Email              string `json:"email"`
	Password           string `json:"password"`
	Expires_in_seconds int    `json:"expires_in_seconds,omitempty"`
}
type User struct {
	ID            uuid.UUID `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Email         string    `json:"email"`
	Token         *string   `json:"token,omitempty"`
	Refresh_token *string   `json:"refresh_token,omitempty"`
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
type tokenstruct struct {
	Token string `json:"token"`
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

func getchirps(w http.ResponseWriter, code int, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	arr := []chirpsOutput{}
	chirps, err := apiconfig.dbQueries.GetChirps(r.Context())
	if err != nil {
		returnwitherror(w, 500, "Could not get chirps")
	}
	for _, v := range chirps {
		arr = append(arr, chirpsOutput{ID: v.ID, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt, Body: v.Body, UserID: v.UserID})
	}
	arrjson, err := json.Marshal(arr)
	if err != nil {
		returnwitherror(w, 500, "Could Not Marshall Chirps")
	}
	w.WriteHeader(code)
	w.Write(arrjson)
}

func returnUser(w http.ResponseWriter, code int, userquery database.User, r *http.Request) {
	token, err := auth.MakeJWT(userquery.ID, apiconfig.jwt_Secret)
	if err != nil {
		returnwitherror(w, 500, "Could not make jwt")
		return
	}
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		returnwitherror(w, 500, "Could not make refresh token")
	}
	userstruct := User{ID: userquery.ID, CreatedAt: userquery.CreatedAt, UpdatedAt: userquery.UpdatedAt, Email: userquery.Email, Token: &token, Refresh_token: &refreshToken}
	_, err = apiconfig.dbQueries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{Token: *userstruct.Refresh_token, UserID: userstruct.ID})
	if err != nil {
		returnwitherror(w, 500, "Could not save refresh token")
		return
	}
	userjson, err := json.Marshal(userstruct)
	if err != nil {
		returnwitherror(w, 500, "Could not marshall userstruct")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(userjson)
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return
	}
	mux := http.NewServeMux()
	apiconfig = &apiConfig{dbQueries: database.New(db), platform: os.Getenv("PLATFORM"), jwt_Secret: os.Getenv("jwt_Secret")}
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
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			returnwitherror(w, 400, "Could Not Find Token")
		}
		params := chirpsInput{}
		var check = 0
		decoderr := decoder.Decode(&params)
		if decoderr != nil {
			check = returnwitherror(w, 500, "Something went wrong")
		}
		if len(params.Body) > 140 {
			check = returnwitherror(w, 400, "Chirp is too long")
		}
		jwt_userid, err := auth.ValidateJWT(token, apiconfig.jwt_Secret)
		params.UserID = jwt_userid
		if err != nil {
			check = returnwitherror(w, 401, "Jwt could not be validated")
		}
		if check == 0 {
			createChirp(w, 201, params, r)
		}
	})
	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		getchirps(w, 200, r)
	})
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		params1 := emailquery{}
		err := decoder.Decode(&params1)
		if err != nil {
			returnwitherror(w, 500, "Something went wrong")
			return
		}
		hashed_password, err := auth.HashPassword(params1.Password)
		if err != nil {
			returnwitherror(w, 400, "Password Cant Be Hashed")
			return
		}
		user, err := apiconfig.dbQueries.CreateUser(r.Context(), database.CreateUserParams{Email: params1.Email, HashedPassword: hashed_password})
		if err != nil {
			returnwitherror(w, 500, "Could not create User")
			return
		}
		returnUser(w, 201, user, r)
	})
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		params1 := emailquery{}
		err := decoder.Decode(&params1)
		if err != nil {
			returnwitherror(w, 500, "Something went wrong")
			return
		}
		user, err := apiconfig.dbQueries.UserByEmail(r.Context(), params1.Email)
		if (err != nil) || (auth.CheckPasswordHash(params1.Password, user.HashedPassword) != nil) {
			returnwitherror(w, 401, "Incorrect email or password")
			return
		}
		returnUser(w, 200, user, r)
	})
	mux.HandleFunc("GET /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		chirpidstring := r.PathValue("chirpID")
		chirpid, err := uuid.Parse(chirpidstring)
		if err != nil {
			returnwitherror(w, 400, "Invalid ChirpID")
		}
		chirp, err := apiconfig.dbQueries.GetChirp(r.Context(), chirpid)
		if err != nil {
			returnwitherror(w, 500, "Could not get chirps")
		}
		chirpstruct := chirpsOutput{ID: chirp.ID, CreatedAt: chirp.CreatedAt, UpdatedAt: chirp.UpdatedAt, Body: chirp.Body, UserID: chirp.UserID}
		chirpjson, err := json.Marshal(chirpstruct)
		if err != nil {
			returnwitherror(w, 500, "Could not marshall chirp")
		}
		w.WriteHeader(200)
		w.Write(chirpjson)
	})
	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			returnwitherror(w, 400, "No Token Provided")
		}
		tokenquery, err := apiconfig.dbQueries.QueryRefreshToken(r.Context(), token)
		if err != nil {
			returnwitherror(w, 401, "Could not find token / is expired")
			return
		}
		acctoken, err := auth.MakeJWT(tokenquery.UserID, apiconfig.jwt_Secret)
		if err != nil {
			returnwitherror(w, 500, "Could not make jwt")
		}
		accstruct := tokenstruct{Token: acctoken}
		accjson, err := json.Marshal(accstruct)
		if err != nil {
			returnwitherror(w, 500, "Could not marshall json")
		}
		w.WriteHeader(200)
		w.Write(accjson)

	})
	mux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			returnwitherror(w, 400, "No Token Provided")
		}
		_, err = apiconfig.dbQueries.RevokeRefreshToken(r.Context(), token)
		if err != nil {
			returnwitherror(w, 500, "Could not Revoke Token")
		}
		w.WriteHeader(204)
	})
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	server.ListenAndServe()
}
