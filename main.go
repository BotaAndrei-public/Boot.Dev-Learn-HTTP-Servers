package main

import (
	"database/sql"
	"os"

	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/BotaAndrei-public/Boot.Dev-Learn-HTTP-Servers/internal/auth"
	"github.com/BotaAndrei-public/Boot.Dev-Learn-HTTP-Servers/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Structs
// / Struct User
type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

// / Struct Chirp
type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

// API-CFG

type apiConfig struct {
	fileserverHits atomic.Int32
	DB             *database.Queries
	platform       string
}

// Methods for API GFG - Handlers
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html")

	w.Write([]byte(fmt.Sprintf(`
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>
		`, cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {

	if cfg.platform != "dev" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": "Forbidden!"}`))
		return
	}
	// Set count users to 0
	cfg.fileserverHits.Store(0)

	err := cfg.DB.ResetUsers(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Colud not reset users"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// CreateUser
func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var params_send database.CreateUserParams
	params := parameters{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid request payload"}`))
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error":"%v"}`, err)))
		return
	}

	params_send.Email = params.Email
	params_send.HashedPassword = hashedPassword

	dbUser, err := cfg.DB.CreateUser(r.Context(), params_send)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error": "Could not createa user: %v"}`, err)))
		return
	}

	userResponse := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}

	data, err := json.Marshal(userResponse)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error:" "Could not marshal response: %v"}`, err)))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}

// Login User TODO
func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error": "%v"}`, err)))
		return
	}

	dbUser, err := cfg.DB.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Incorrect email or password"}`))
		return // <-- AICI LIPSEA return!
	}

	match, err := auth.CheckPasswordHash(params.Password, dbUser.HashedPassword)
	if err != nil || !match {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Incorrect email or password"}`))
		return
	}

	responseUser := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}

	data, err := json.Marshal(responseUser)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func mainHandler(w http.ResponseWriter, req *http.Request) {
	//if req.URL.Path != "/"{
	//	http.NotFound(w, req)
	//	return
	//}
	//http.ServeFile(w, req, "index.html")

	//if req.URL.Path != "/"{
	//	http.Redirect(w, req, "/", http.StatusSeeOther)
	//	return
	//}

	http.FileServer(http.Dir(".")).ServeHTTP(w, req)
}

func ReadinessHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Universal Interface ResponseWriter
type IRW interface {
	ResponseWR(w http.ResponseWriter, options ...any)
}

// Universal Handler ResponseWriter
type HRW struct {
	ErrBody      string `json:"error,omitempty"`
	TextBody     string `json:"value"`
	Valid        bool   `json:"valid,omitempty"`
	Cleaned_body string `json:"cleaned_body,omitempty"`
}

//GET By ID Chirp from Chirps

func AutoResponseWR(w http.ResponseWriter, R IRW, options ...any) {
	R.ResponseWR(w, options...)
}

// GET Response as hrw
func (hrw HRW) ResponseWR(w http.ResponseWriter, options ...any) {
	w.Header().Set("Content-Type", "application/json")
	if hrw.Valid {
		w.WriteHeader(200)

	} else {
		w.WriteHeader(400)

	}
	dat, err := json.Marshal(hrw)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}
	w.Write(dat)

}

// POST Create
// FROM Chirps
//func (c Chirp) ResponseWR(w http.ResponseWriter) {

//	dat, err := json.Marshal(c)
//	if err != nil {
//		w.Header().Set("content-Type", "application/json")
//		w.WriteHeader(500)
//		return
//	}

//ALL GOOD in the HOOD
//	w.Header().Set("Content-Type", "application/json")
//	w.WriteHeader(http.StatusCreated)
//	w.Write(dat)

//}

// GET/POST/PUT/DELETE GENERIC RESPONSE - SETTABLE BAD/GOOD HEADER & ERROR
func (c Chirp) ResponseWR(w http.ResponseWriter, options ...any) {
	w.Header().Set("Content-Type", "application/json")

	goodHeader := 0
	badHeader := 0
	var customErr error

	for _, opt := range options {
		switch v := opt.(type) {

		case int:
			if v >= 200 && v < 300 {
				goodHeader = v
			} else if v >= 400 {
				badHeader = v
			}
		case error:
			customErr = v
		case string:
			customErr = fmt.Errorf("%s", v)
		}
	}

	dat, err := json.Marshal(c)
	if err != nil {
		if badHeader == 0 {
			badHeader = 500
		}
		w.WriteHeader(badHeader)
		if customErr != nil {
			w.Write([]byte(fmt.Sprintf(`{"error":"%v"}`, customErr)))
		} else {
			w.Write([]byte(fmt.Sprintf(`{"error":"%v"}`, err)))
		}
		return
	}
	if goodHeader == 0 {
		goodHeader = 200
	}
	w.WriteHeader(goodHeader)
	w.Write(dat)
	return
}

// GET All Chirps
type listChitps []Chirp

func (c listChitps) ResponseWR(w http.ResponseWriter, options ...any) {
	data, err := json.Marshal(c)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Could not marshal chirps"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)

}

// GET chirp by ID
func (cfg *apiConfig) getChirpByID(w http.ResponseWriter, r *http.Request) {

	chirpIDString := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDString)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf(`{"error": "Invalid chirp ID: %v"}`, err)))
		return
	}

	dbChirps, err := cfg.DB.GetChirp(r.Context(), chirpID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf(`{"error:": "Invalid chirp ID: %v"}`, err)))
		return
	}

	foudChirp := Chirp{
		ID:        dbChirps.ID,
		CreatedAt: dbChirps.CreatedAt,
		UpdatedAt: dbChirps.UpdatedAt,
		Body:      dbChirps.Body,
		UserID:    dbChirps.UserID,
	}

	AutoResponseWR(w, foudChirp, 404)

}

// GET chirps
func (cfg *apiConfig) getChirps(w http.ResponseWriter, r *http.Request) {
	dbChirps, err := cfg.DB.GetChirps(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"error:" "Could not retrieve chirps: %v"}`, err)))
		return
	}
	chirps := []Chirp{}
	for _, c_next := range dbChirps {
		chirp := Chirp{
			ID:        c_next.ID,
			CreatedAt: c_next.CreatedAt,
			UpdatedAt: c_next.UpdatedAt,
			Body:      c_next.Body,
			UserID:    c_next.UserID,
		}
		chirps = append(chirps, chirp)
	}

	AutoResponseWR(w, listChitps(chirps))
	return

}

// POST chirps
func (cfg *apiConfig) chirpHandler(w http.ResponseWriter, req *http.Request) {

	//General struct for Json
	type JsonBody struct {
		Body string    `json:"body"`
		ID   uuid.UUID `json:"user_id"`
	}

	//Internal params / func helper
	// HelpParamsFunc - HPF
	type HPF struct {
		MaxChar  int
		BanWords []string
	}

	//funcs HPF
	getMaxChar := func(hpf HPF) int {
		if hpf.MaxChar == 0 {
			return 140
		}
		return hpf.MaxChar
	}
	filterByKeywords := func(msg *string, filterList *[]string) string {
		var splitList, c_splitList []string
		var c_msg string
		c_msg = *msg
		c_splitList = strings.Split(*msg, " ")
		c_msg = strings.ToLower(c_msg)
		splitList = strings.Split(c_msg, " ")

		for _, w := range *filterList {
			for i, bw := range splitList {
				if bw == w {
					c_splitList[i] = "****"
				}
			}
			//Print line by line
			//fmt.Printf("#%v\n", splitList)
			*msg = strings.Join(c_splitList, " ")
		}
		return ""
	}

	//func
	var data JsonBody
	var hpf HPF
	hpf.BanWords = []string{"kerfuffle", "sharbert", "fornax"}

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&data)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		return
	}

	//Validate length
	if len(data.Body) > getMaxChar(hpf) {
		resp := HRW{ErrBody: "error", TextBody: "Chirp is too long", Valid: false}
		AutoResponseWR(w, resp)
		return
	}

	//id Valid is TRUE
	filterByKeywords(&data.Body, &hpf.BanWords)

	//Save in DB
	var params database.CreateChirpParams
	params.Body = data.Body
	params.UserID = data.ID

	dbChirp, err := cfg.DB.CreateChirp(req.Context(), params)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"error": "Could not save chirp: %v"}`, err)))
		return
	}

	chirpResponse := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	AutoResponseWR(w, chirpResponse, http.StatusCreated)
	return

	//OLD way
	//resp := HRW{ErrBody: "Valid", TextBody: "true", Cleaned_body: data.Body, Valid: true}
	//AutoResponseWR(w, resp)

}

func JsonTestHandler(w http.ResponseWriter, req *http.Request) {

	type TestBody struct {
		Body string `json:"body"`
	}

	var data TestBody

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&data)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error": "%v"}`, err)))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(data.Body))
	fmt.Printf("Data: \n%v", data)
}

func main() {
	godotenv.Load()

	//DB
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Error DB: %v", err)
	}

	dbQueries := database.New(db)

	cfg := &apiConfig{
		DB:       dbQueries,
		platform: os.Getenv("PLATFORM"),
	}

	mux := http.NewServeMux()
	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.HandlerFunc(mainHandler))))
	// You can just use : "GET /api/X" for taht, but the current state of the code is more... fancier? mb?
	mux.Handle("GET /api/healthz", http.StripPrefix("/api", http.HandlerFunc(ReadinessHandler)))
	s := &http.Server{
		Addr: ":8080",

		Handler: mux,
	}
	mux.HandleFunc("GET /admin/metrics", cfg.metricsHandler)
	mux.HandleFunc("POST /admin/reset", cfg.resetHandler)
	mux.HandleFunc(" /api/chirps", cfg.chirpHandler)
	mux.HandleFunc("POST /api/users", cfg.createUser)
	mux.HandleFunc("GET /api/chirps", cfg.getChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.getChirpByID)
	mux.HandleFunc("POST /api/login", cfg.loginHandler)
	fmt.Println("Server is running on port 8080...")

	log.Fatal(s.ListenAndServe())
}
