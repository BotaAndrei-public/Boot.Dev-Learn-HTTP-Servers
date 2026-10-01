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

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid request payload"}`))
		return
	}

	dbUser, err := cfg.DB.CreateUser(r.Context(), params.Email)
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
	ResponseWR(w http.ResponseWriter)
}

// Universal Handler ResponseWriter
type HRW struct {
	ErrBody      string `json:"error,omitempty"`
	TextBody     string `json:"value"`
	Valid        bool   `json:"valid,omitempty"`
	Cleaned_body string `json:"cleaned_body,omitempty"`
}

func (hrw HRW) ResponseWR(w http.ResponseWriter) {
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

func AutoResponseWR(w http.ResponseWriter, R IRW) {
	R.ResponseWR(w)
}

func Chirp(w http.ResponseWriter, req *http.Request) {

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
	if len(data.Body) > getMaxChar(hpf) {
		resp := HRW{ErrBody: "error", TextBody: "Chirp is too long", Valid: false}
		AutoResponseWR(w, resp)
	} else {
		filterByKeywords(&data.Body, &hpf.BanWords)
		resp := HRW{ErrBody: "Valid", TextBody: "true", Cleaned_body: data.Body, Valid: true}
		AutoResponseWR(w, resp)
	}

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
	mux.HandleFunc(" /api/chirps", Chirp)
	mux.HandleFunc("POST /api/users", cfg.createUser)
	fmt.Println("Server is running on port 8080...")

	log.Fatal(s.ListenAndServe())
}
