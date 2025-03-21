package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
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
	polka_key      string
}
type errordata struct {
	Error string `json:"error"`
}
type emailquery struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type User struct {
	ID            uuid.UUID `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Email         string    `json:"email"`
	Token         *string   `json:"token,omitempty"`
	Refresh_token *string   `json:"refresh_token,omitempty"`
	Is_chirpy_red bool      `json:"is_chirpy_red"`
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
type polkaInput struct {
	Event string `json:"event"`
	Data  struct {
		UserID uuid.UUID `json:"user_id"`
	} `json:"data"`
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
		return
	}
	chirpresp := chirpsOutput{ID: chirp.ID, CreatedAt: chirp.CreatedAt, UpdatedAt: chirp.UpdatedAt, Body: chirp.Body, UserID: chirp.UserID}
	rspjson, err := json.Marshal(chirpresp)
	if err != nil {
		returnwitherror(w, 500, "Could not marshall chirpresp")
		return
	}
	w.WriteHeader(code)
	w.Write(rspjson)
}

func getchirps(w http.ResponseWriter, code int, r *http.Request, authorID string, sortvalue string) {
	w.Header().Set("Content-Type", "application/json")
	arr := []chirpsOutput{}
	var chirps []database.Chirp
	var err error
	if authorID != "" {
		suuid, err := uuid.Parse(authorID)
		if err != nil {
			returnwitherror(w, 400, "Could not find UserID")
			return
		}
		chirps, err = apiconfig.dbQueries.GetChirpsByAuthor(r.Context(), suuid)
		if err != nil {
			returnwitherror(w, 500, "Could not get chirps")
			return
		}
	} else {
		chirps, err = apiconfig.dbQueries.GetChirps(r.Context())
		if err != nil {
			returnwitherror(w, 500, "Could not get chirps")
			return
		}
	}
	for _, v := range chirps {
		arr = append(arr, chirpsOutput{ID: v.ID, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt, Body: v.Body, UserID: v.UserID})
	}

	var ascending bool
	if sortvalue == "ASC" {
		ascending = true
	} else {
		ascending = false
	}

	sort.SliceStable(arr, func(i, j int) bool {
		if ascending {
			return arr[i].CreatedAt.Before(arr[j].CreatedAt)
		}
		return arr[i].CreatedAt.After(arr[j].CreatedAt)
	})

	arrjson, err := json.Marshal(arr)
	if err != nil {
		returnwitherror(w, 500, "Could Not Marshall Chirps")
		return
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
		return
	}
	userstruct := User{ID: userquery.ID, CreatedAt: userquery.CreatedAt, UpdatedAt: userquery.UpdatedAt, Email: userquery.Email, Token: &token, Refresh_token: &refreshToken, Is_chirpy_red: userquery.IsChirpyRed}
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
	apiconfig = &apiConfig{dbQueries: database.New(db), platform: os.Getenv("PLATFORM"), jwt_Secret: os.Getenv("jwt_Secret"), polka_key: os.Getenv("POLKA_KEY")}
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
			return
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
		authorID := r.URL.Query().Get("author_id")
		sort := r.URL.Query().Get("sort")
		if (sort == "") || (sort == "asc") {
			sort = "ASC"
		} else {
			sort = "DESC"
		}
		getchirps(w, 200, r, authorID, sort)
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
			return
		}
		chirp, err := apiconfig.dbQueries.GetChirp(r.Context(), chirpid)
		if err != nil {
			returnwitherror(w, 404, "Could not get chirps")
			return
		}
		chirpstruct := chirpsOutput{ID: chirp.ID, CreatedAt: chirp.CreatedAt, UpdatedAt: chirp.UpdatedAt, Body: chirp.Body, UserID: chirp.UserID}
		chirpjson, err := json.Marshal(chirpstruct)
		if err != nil {
			returnwitherror(w, 500, "Could not marshall chirp")
			return
		}
		w.WriteHeader(200)
		w.Write(chirpjson)
	})
	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			returnwitherror(w, 400, "No Token Provided")
			return
		}
		tokenquery, err := apiconfig.dbQueries.QueryRefreshToken(r.Context(), token)
		if err != nil {
			returnwitherror(w, 401, "Could not find token / is expired")
			return
		}
		acctoken, err := auth.MakeJWT(tokenquery.UserID, apiconfig.jwt_Secret)
		if err != nil {
			returnwitherror(w, 500, "Could not make jwt")
			return
		}
		accstruct := tokenstruct{Token: acctoken}
		accjson, err := json.Marshal(accstruct)
		if err != nil {
			returnwitherror(w, 500, "Could not marshall json")
			return
		}
		w.WriteHeader(200)
		w.Write(accjson)

	})
	mux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			returnwitherror(w, 400, "No Token Provided")
			return
		}
		_, err = apiconfig.dbQueries.RevokeRefreshToken(r.Context(), token)
		if err != nil {
			returnwitherror(w, 500, "Could not Revoke Token")
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("PUT /api/users", func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			returnwitherror(w, 401, "No token Provided")
			return
		}
		params := emailquery{}
		err = decoder.Decode(&params)
		if err != nil {
			returnwitherror(w, 400, "could not decode body")
			return
		}
		tokenID, err := auth.ValidateJWT(token, apiconfig.jwt_Secret)
		if err != nil {
			returnwitherror(w, 401, "There is a problem with your token")
			return
		}
		hashedpsw, err := auth.HashPassword(params.Password)
		if err != nil {
			returnwitherror(w, 500, "could not hash password")
			return
		}
		qres, err := apiconfig.dbQueries.ChangePassword(r.Context(), database.ChangePasswordParams{HashedPassword: hashedpsw, ID: tokenID})
		if err != nil {
			returnwitherror(w, 500, "password change failed")
			return
		}
		_ = qres
		params.Password = hashedpsw
		paramsjson, err := json.Marshal(params)
		if err != nil {
			returnwitherror(w, 500, "could not marshall return parameters")
			return
		}
		w.WriteHeader(200)
		w.Write(paramsjson)
	})
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			returnwitherror(w, 401, "No token Provided")
			return
		}
		chirpidstring := r.PathValue("chirpID")
		chirpid, err := uuid.Parse(chirpidstring)
		if err != nil {
			returnwitherror(w, 400, "Invalid ChirpID")
			return
		}
		chirp, err := apiconfig.dbQueries.GetChirp(r.Context(), chirpid)
		if err != nil {
			returnwitherror(w, 404, "Could not get chirps")
			return
		}
		chirpstruct := chirpsOutput{ID: chirp.ID, CreatedAt: chirp.CreatedAt, UpdatedAt: chirp.UpdatedAt, Body: chirp.Body, UserID: chirp.UserID}
		tokenid, err := auth.ValidateJWT(token, apiconfig.jwt_Secret)
		if err != nil {
			returnwitherror(w, 400, "Token could not be verified")
			return
		}
		if chirpstruct.UserID == tokenid {
			err = apiconfig.dbQueries.DeleteChirp(r.Context(), chirpstruct.ID)
			if err != nil {
				returnwitherror(w, 500, "Could not delete chirp")
				return
			}
			w.WriteHeader(204)
		}
		w.WriteHeader(403)
	})
	mux.HandleFunc("POST /api/polka/webhooks", func(w http.ResponseWriter, r *http.Request) {
		apikey, err := auth.GetAPIKey(r.Header)
		if (err != nil) || (apikey != apiconfig.polka_key) {
			returnwitherror(w, 401, "Wrong API Key")
			return
		}
		decoder := json.NewDecoder(r.Body)
		params := polkaInput{}
		err = decoder.Decode(&params)
		if err != nil {
			returnwitherror(w, 400, "could not decode body")
			return
		}
		if params.Event != "user.upgraded" {
			w.WriteHeader(204)
			return
		}
		qresult, err := apiconfig.dbQueries.UpgradeUser(r.Context(), params.Data.UserID)
		if err != nil {
			returnwitherror(w, 404, "Could not find user")
			return
		}
		_ = qresult
		w.WriteHeader(204)
	})
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	server.ListenAndServe()
}
