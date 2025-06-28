package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
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
		rw.Header().Add("Content-Type", "text/html")

		rw.WriteHeader(200)
		fmt.Fprintf(rw,
			`<html>
	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited %d times!</p>
	</body>
</html>`, counter)
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

	mux.Handle("/app/", (&cfg).middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))

	mux.Handle("/admin/metrics", (&cfg).middlewareMetricsReader())

	mux.Handle("/admin/reset", (&cfg).middlewareMetricsReset())

	mux.HandleFunc("/api/healthz", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add("Content-Type", "text/plain; charset=utf-8")
		rw.WriteHeader(200)
		fmt.Fprint(rw, "OK")
	})

	mux.HandleFunc("POST /api/validate_chirp", func(rw http.ResponseWriter, req *http.Request) {
		type parameters struct {
			Body string `json:"body"`
		}

		type validReturnVals struct {
			Cleaned_Body string `json:"cleaned_body"`
		}

		type errorReturnVals struct {
			Error string `json:"error"`
		}

		decoder := json.NewDecoder(req.Body)
		params := parameters{}
		parseErr := decoder.Decode(&params)

		rw.Header().Set("Content-Type", "application/json")

		if parseErr != nil {
			resp := errorReturnVals{
				Error: parseErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(400)
			rw.Write(data)
			return
		}

		valid := len(params.Body) < 141
		if !valid {
			resp := errorReturnVals{
				Error: "Chirp is too long",
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(400)
			rw.Write(data)
			return
		}

		badWords := []string{"kerfuffle", "sharbert", "fornax"}

		message := strings.Fields(params.Body)

		cleaned := ""

		for _, word := range message {
			if !slices.Contains(badWords, strings.ToLower(word)) {
				if cleaned != "" {
					cleaned += " "
				}
				cleaned += word
			} else {
				if cleaned != "" {
					cleaned += " "
				}
				cleaned += "****"
			}
		}

		resp := validReturnVals{
			Cleaned_Body: cleaned,
		}

		data, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			rw.WriteHeader(500)
			return
		}

		rw.WriteHeader(200)
		rw.Write(data)
	})

	var server http.Server
	server.Handler = mux
	server.Addr = ":8080"

	server.ListenAndServe()
}
