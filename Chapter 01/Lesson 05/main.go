package main

import "net/http"

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(".")))

	var server http.Server
	server.Handler = mux
	server.Addr = ":8080"

	server.ListenAndServe()
}
