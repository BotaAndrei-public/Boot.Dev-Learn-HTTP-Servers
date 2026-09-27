package main

import(
	"fmt"
	"net/http"
	"log"
)

func mainHandler(w http.ResponseWriter, req *http.Request){
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


func main(){
	mux := http.NewServeMux()
	mux.HandleFunc("/", mainHandler)
	
	s := &http.Server{
		Addr: ":8080",
		Handler: mux,
	}
	
	fmt.Println("Server is running on port 8080...")

	log.Fatal(s.ListenAndServe())
}
