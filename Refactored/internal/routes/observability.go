package routes

import (
	"fmt"
	"net/http"

	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/config"
)

func MapObservabilityRoutes(mux *http.ServeMux, cfg *config.AppConfig) {
	MapGet(mux, "/admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		counter := cfg.FileServerHits.Load()
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(200)
		fmt.Fprintf(w,
			`<html>
	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited %d times!</p>
	</body>
</html>`, counter)
	})

	MapPost(mux, "/admin/reset", func(w http.ResponseWriter, r *http.Request) {
		cfg.ResetApp()
	})

	MapGet(mux, "/api/healthz", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add("Content-Type", "text/plain; charset=utf-8")
		rw.WriteHeader(200)
		fmt.Fprint(rw, "OK")
	})
}
