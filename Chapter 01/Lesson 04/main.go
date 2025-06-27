package main

import "net/http"

func main() {
	mux := http.NewServeMux()

	var server http.Server
	server.Handler = mux
	server.Addr = ":8080"

	server.ListenAndServe()
}
