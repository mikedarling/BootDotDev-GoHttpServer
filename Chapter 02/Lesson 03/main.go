package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(rw, req)
	})
}

func (cfg *apiConfig) middlewareMetricsReader() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		counter := cfg.fileserverHits.Load()
		rw.Header().Add("Content-Type", "text/plain; charset=utf-8")
		rw.WriteHeader(200)
		fmt.Fprintf(rw, "Hits: %v", counter)
	})
}

func (cfg *apiConfig) middlewareMetricsReset() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Store(0)
		rw.WriteHeader(200)
	})
}

func main() {
	mux := http.NewServeMux()
	var cfg apiConfig

	mux.HandleFunc("GET /healthz", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add("Content-Type", "text/plain; charset=utf-8")
		rw.WriteHeader(200)
		fmt.Fprint(rw, "OK")
	})

	mux.Handle("GET /metrics", (&cfg).middlewareMetricsReader())

	mux.Handle("POST /reset", (&cfg).middlewareMetricsReset())

	mux.Handle("/app/", (&cfg).middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))

	var server http.Server
	server.Handler = mux
	server.Addr = ":8080"

	server.ListenAndServe()
}
