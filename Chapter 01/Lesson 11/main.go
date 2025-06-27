package main

import "net/http"

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add("Content-Type", "text/plain; charset=utf-8")
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))
	})

	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))

	var server http.Server
	server.Handler = mux
	server.Addr = ":8080"

	server.ListenAndServe()
}
