package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      database.Queries
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
		if os.Getenv("PLATFORM") != "dev" {
			rw.WriteHeader(403)
			return
		}

		cfg.dbQueries.DeleteUsers(req.Context())

		cfg.fileserverHits.Store(0)
		rw.WriteHeader(200)
	})
}

func main() {
	godotenv.Load()
	var cfg apiConfig

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		fmt.Print("Could not connect to database.")
	}

	cfg.dbQueries = *database.New(db)

	mux := http.NewServeMux()

	mux.Handle("GET /app", (&cfg).middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))

	mux.Handle("GET /admin/metrics", (&cfg).middlewareMetricsReader())

	mux.Handle("POST /admin/reset", (&cfg).middlewareMetricsReset())

	mux.HandleFunc("POST /api/chirps", func(rw http.ResponseWriter, req *http.Request) {
		type parameters struct {
			Body   string    `json:"body"`
			UserId uuid.UUID `json:"user_id"`
		}

		type errorReturnVals struct {
			Error string `json:"error"`
		}

		type chripReturnVals struct {
			ID        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserId    uuid.UUID `json:"user_id"`
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

		queryParams := database.CreateChirpsParams{
			Body:   params.Body,
			UserID: params.UserId,
		}

		chirp, dbErr := cfg.dbQueries.CreateChirps(req.Context(), queryParams)

		if dbErr != nil {
			resp := errorReturnVals{
				Error: dbErr.Error(),
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

		resp := chripReturnVals{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserId:    chirp.UserID,
		}

		data, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			rw.WriteHeader(500)
			return
		}

		rw.WriteHeader(201)
		rw.Write(data)
	})

	mux.HandleFunc("GET /api/healthz", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add("Content-Type", "text/plain; charset=utf-8")
		rw.WriteHeader(200)
		fmt.Fprint(rw, "OK")
	})

	mux.HandleFunc("POST /api/users", func(rw http.ResponseWriter, req *http.Request) {
		type parameters struct {
			Email string `json:"email"`
		}

		type errorReturnVals struct {
			Error string `json:"error"`
		}

		type userReturnVals struct {
			ID        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Email     string    `json:"email"`
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

		user, dbErr := cfg.dbQueries.CreateUser(req.Context(), params.Email)

		if dbErr != nil {
			resp := errorReturnVals{
				Error: dbErr.Error(),
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

		resp := userReturnVals{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
		}

		data, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			rw.WriteHeader(500)
			return
		}

		rw.WriteHeader(201)
		rw.Write(data)
	})

	var server http.Server
	server.Handler = mux
	server.Addr = ":8080"

	server.ListenAndServe()
}
