package config

import (
	"errors"
	"net/http"
)

func (cfg *AppConfig) IncrementServerHits(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		cfg.FileServerHits.Add(1)
		next.ServeHTTP(rw, req)
	})
}

func (cfg *AppConfig) ResetApp(r *http.Request) error {
	if cfg.Platform != "dev" {
		return errors.New("operation not allowed on this environment")
	}

	deleteUsersErr := cfg.DbQueries.DeleteUsers(r.Context())
	if deleteUsersErr != nil {
		return deleteUsersErr
	}

	cfg.FileServerHits.Store(0)
	return nil
}
