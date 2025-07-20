package main

import (
	"fmt"
	"net/http"

	_ "github.com/lib/pq"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/config"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/routes"
)

func main() {
	cfg, cfgErr := config.LoadAppConfig()
	if cfgErr != nil {
		fmt.Printf("%v\n", cfgErr.Error())
		return
	}

	mux := http.NewServeMux()

	mux.Handle("GET /app", (&cfg).IncrementServerHits(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))

	routes.MapChirpRoutes(mux, &cfg)

	routes.MapAuthRoutes(mux, &cfg)

	routes.MapUserRoutes(mux, &cfg)

	routes.MapObservabilityRoutes(mux, &cfg)

	var server http.Server
	server.Handler = mux
	server.Addr = ":8080"

	server.ListenAndServe()
}
