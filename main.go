package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

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
	cfg.fileserverHits.Store(0)
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

func main() {

	cfg := &apiConfig{}

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

	fmt.Println("Server is running on port 8080...")

	log.Fatal(s.ListenAndServe())
}
