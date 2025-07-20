package config

import (
	"net/http"
	"os"
)

func (cfg *AppConfig) IncrementServerHits(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		cfg.FileServerHits.Add(1)
		next.ServeHTTP(rw, req)
	})
}

func (cfg *AppConfig) ResetApp() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if os.Getenv("PLATFORM") != "dev" {
			rw.WriteHeader(403)
			return
		}

		cfg.DbQueries.DeleteUsers(req.Context())

		cfg.FileServerHits.Store(0)
		rw.WriteHeader(200)
	})
}
