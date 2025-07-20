package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/auth"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      database.Queries
	jwtSecret      string
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

	cfg.jwtSecret = os.Getenv("JWT_SECRET")

	cfg.dbQueries = *database.New(db)

	mux := http.NewServeMux()

	mux.Handle("GET /app", (&cfg).middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))

	mux.Handle("GET /admin/metrics", (&cfg).middlewareMetricsReader())

	mux.Handle("POST /admin/reset", (&cfg).middlewareMetricsReset())

	mux.HandleFunc("GET /api/chirps", func(rw http.ResponseWriter, req *http.Request) {
		type chripReturnVals struct {
			ID        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserId    uuid.UUID `json:"user_id"`
		}

		type errorReturnVals struct {
			Error string `json:"error"`
		}

		queryParams := req.URL.Query()
		authorId := queryParams.Get("author_id")
		var userId uuid.UUID
		var userIdParseErr error
		if authorId != "" {
			userId, userIdParseErr = uuid.Parse(authorId)
			if userIdParseErr != nil {
				resp := errorReturnVals{
					Error: userIdParseErr.Error(),
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
		}

		var chirps []database.Chirp
		var dbErr error

		if uuid.Nil != userId {
			chirps, dbErr = cfg.dbQueries.GetChirpsByUser(req.Context(), userId)
		} else {
			chirps, dbErr = cfg.dbQueries.GetAllChirps(req.Context())
		}

		sortOrder := queryParams.Get("sort")
		if sortOrder != "" && sortOrder == "desc" {
			sort.Slice(chirps, func(i, j int) bool { return chirps[j].CreatedAt.Before(chirps[i].CreatedAt) })
		}

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

		resp := make([]chripReturnVals, len(chirps))

		for i, chirp := range chirps {
			resp[i] = chripReturnVals{
				ID:        chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserId:    chirp.UserID,
			}
		}

		data, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			rw.WriteHeader(500)
			return
		}

		rw.WriteHeader(200)
		rw.Write(data)
	})

	mux.HandleFunc("GET /api/chirps/{chirpID}", func(rw http.ResponseWriter, req *http.Request) {
		type chripReturnVals struct {
			ID        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserId    uuid.UUID `json:"user_id"`
		}

		type errorReturnVals struct {
			Error string `json:"error"`
		}

		chirpId, uuidParseErr := uuid.Parse(req.PathValue("chirpID"))
		if uuidParseErr != nil {
			resp := errorReturnVals{
				Error: uuidParseErr.Error(),
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

		chirp, dbErr := cfg.dbQueries.GetChirp(req.Context(), chirpId)
		if dbErr != nil {
			resp := errorReturnVals{
				Error: dbErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(404)
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

		rw.WriteHeader(200)
		rw.Write(data)
	})

	mux.HandleFunc("DELETE /api/chirps/{chirpID}", func(rw http.ResponseWriter, req *http.Request) {
		type errorReturnVals struct {
			Error string `json:"error"`
		}

		rw.Header().Set("Content-Type", "application/json")

		token, tokenErr := auth.GetBearerToken(req.Header)
		if tokenErr != nil {
			resp := errorReturnVals{
				Error: tokenErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(401)
			rw.Write(data)
			return
		}

		userId, validateErr := auth.ValidateJWT(token, cfg.jwtSecret)
		if validateErr != nil {
			resp := errorReturnVals{
				Error: validateErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(401)
			rw.Write(data)
			return
		}

		chirpId, uuidParseErr := uuid.Parse(req.PathValue("chirpID"))
		if uuidParseErr != nil {
			resp := errorReturnVals{
				Error: uuidParseErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(404)
			rw.Write(data)
			return
		}

		chirp, chirpDbErr := cfg.dbQueries.GetChirpById(req.Context(), chirpId)
		if chirpDbErr != nil {
			resp := errorReturnVals{
				Error: chirpDbErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(403)
			rw.Write(data)
			return
		}

		if chirp.UserID != userId {
			rw.WriteHeader(403)
			return
		}

		delChirpErr := cfg.dbQueries.DeleteChirpById(req.Context(), chirp.ID)
		if delChirpErr != nil {
			resp := errorReturnVals{
				Error: delChirpErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(403)
			rw.Write(data)
			return
		}

		rw.WriteHeader(204)
	})

	mux.HandleFunc("POST /api/chirps", func(rw http.ResponseWriter, req *http.Request) {
		type parameters struct {
			Body string `json:"body"`
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

		rw.Header().Set("Content-Type", "application/json")

		token, tokenErr := auth.GetBearerToken(req.Header)
		if tokenErr != nil {
			resp := errorReturnVals{
				Error: tokenErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(401)
			rw.Write(data)
			return
		}

		userId, validateErr := auth.ValidateJWT(token, cfg.jwtSecret)
		if validateErr != nil {
			resp := errorReturnVals{
				Error: validateErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(401)
			rw.Write(data)
			return
		}

		decoder := json.NewDecoder(req.Body)
		params := parameters{}
		parseErr := decoder.Decode(&params)
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
			UserID: userId,
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

	mux.HandleFunc("POST /api/login", func(rw http.ResponseWriter, req *http.Request) {
		type parameters struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		type errorReturnVals struct {
			Error string `json:"error"`
		}

		type loginReturnVals struct {
			ID           uuid.UUID `json:"id"`
			CreatedAt    time.Time `json:"created_at"`
			UpdatedAt    time.Time `json:"updated_at"`
			Email        string    `json:"email"`
			Token        string    `json:"token"`
			RefreshToken string    `json:"refresh_token"`
			IsChirpyRed  bool      `json:"is_chirpy_red"`
		}

		rw.Header().Set("Content-Type", "application/json")

		decoder := json.NewDecoder(req.Body)
		params := parameters{}
		parseErr := decoder.Decode(&params)
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

		user, dbErr := cfg.dbQueries.GetUserByEmail(req.Context(), params.Email)
		if dbErr != nil {
			rw.WriteHeader(401)
			rw.Write([]byte("dbErr: Incorrect email or password"))
			return
		}

		if params.Email != user.Email {
			rw.WriteHeader(401)
			rw.Write([]byte("user email does not match: Incorrect email or password"))
			return
		}

		hasingErr := auth.CheckPasswordHash(params.Password, user.HashedPassword)
		if hasingErr != nil {
			rw.WriteHeader(401)
			rw.Write([]byte("Check Password Hash: Incorrect email or password"))
			return
		}

		token, tokenErr := auth.MakeJWT(user.ID, cfg.jwtSecret, 60*time.Second)
		if tokenErr != nil {
			rw.WriteHeader(500)
			rw.Write([]byte("Could not create JWT."))
			return
		}

		refreshToken, refreshTokenErr := auth.MakeRefreshToken()
		if refreshTokenErr != nil {
			rw.WriteHeader(500)
			rw.Write([]byte("Could not create Refresh Token."))
			return
		}

		refreshExpiration := time.Now().UTC().Add(time.Hour * 24 * 60)

		queryParams := database.SaveTokenParams{
			Token:     refreshToken,
			UserID:    user.ID,
			ExpiresAt: refreshExpiration,
		}

		_, dbRefreshTokenErr := cfg.dbQueries.SaveToken(req.Context(), queryParams)
		if dbRefreshTokenErr != nil {
			rw.WriteHeader(500)
			rw.Write([]byte("Could not create Refresh Token."))
			return
		}

		resp := loginReturnVals{
			ID:           user.ID,
			Email:        user.Email,
			CreatedAt:    user.CreatedAt,
			UpdatedAt:    user.UpdatedAt,
			Token:        token,
			RefreshToken: refreshToken,
			IsChirpyRed:  user.IsChirpyRed,
		}

		data, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			rw.WriteHeader(500)
			return
		}

		rw.WriteHeader(200)
		rw.Write(data)
	})

	mux.HandleFunc("POST /api/polka/webhooks", func(rw http.ResponseWriter, req *http.Request) {
		type parameters struct {
			Event string `json:"event"`
			Data  struct {
				UserId uuid.UUID `json:"user_id"`
			} `json:"data"`
		}

		type errorReturnVals struct {
			Error string `json:"error"`
		}

		rw.Header().Set("Content-Type", "application/json")

		apiKey, apiKeyErr := auth.GetAPIKey(req.Header)
		if apiKeyErr != nil {
			resp := errorReturnVals{
				Error: apiKeyErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(401)
			rw.Write(data)
			return
		}

		if apiKey != os.Getenv("POLKA_KEY") {
			rw.WriteHeader(401)
			return
		}

		decoder := json.NewDecoder(req.Body)
		params := parameters{}
		parseErr := decoder.Decode(&params)
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

		if params.Event != "user.upgraded" {
			rw.WriteHeader(204)
			return
		}

		_, dbErr := cfg.dbQueries.UpgradeUserById(req.Context(), params.Data.UserId)
		if dbErr != nil {
			resp := errorReturnVals{
				Error: dbErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(404)
			rw.Write(data)
			return
		}

		rw.WriteHeader(204)
	})

	mux.HandleFunc("POST /api/refresh", func(rw http.ResponseWriter, req *http.Request) {
		type errorReturnVals struct {
			Error string `json:"error"`
		}

		type tokenReturnVals struct {
			Token string `json:"token"`
		}

		token, tokenErr := auth.GetBearerToken(req.Header)
		if tokenErr != nil {
			resp := errorReturnVals{
				Error: tokenErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(401)
			rw.Write(data)
			return
		}

		tokenRecord, getDbTokenErr := cfg.dbQueries.GetUserFromRefreshToken(req.Context(), token)
		if getDbTokenErr != nil {
			rw.WriteHeader(401)
			return
		}

		if tokenRecord.RevokedAt.Valid && time.Now().UTC().After(tokenRecord.RevokedAt.Time) {
			rw.WriteHeader(401)
			return
		}

		if time.Now().UTC().After(tokenRecord.ExpiresAt) {
			rw.WriteHeader(401)
			return
		}

		bearer, bearerErr := auth.MakeJWT(tokenRecord.UserID, os.Getenv("JWT_SECRET"), time.Hour)
		if bearerErr != nil {
			rw.Write([]byte("Could not create bearer token"))
			rw.WriteHeader(500)
			return
		}

		resp := tokenReturnVals{
			Token: bearer,
		}

		data, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			rw.WriteHeader(500)
			return
		}

		rw.WriteHeader(200)
		rw.Write(data)
	})

	mux.HandleFunc("POST /api/revoke", func(rw http.ResponseWriter, req *http.Request) {
		type errorReturnVals struct {
			Error string `json:"error"`
		}

		token, tokenErr := auth.GetBearerToken(req.Header)
		if tokenErr != nil {
			resp := errorReturnVals{
				Error: tokenErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(401)
			rw.Write(data)
			return
		}

		tokenRecord, getDbTokenErr := cfg.dbQueries.GetUserFromRefreshToken(req.Context(), token)
		if getDbTokenErr != nil {
			rw.WriteHeader(401)
			return
		}

		if tokenRecord.RevokedAt.Valid && time.Now().UTC().After(tokenRecord.RevokedAt.Time) {
			rw.WriteHeader(401)
			return
		}

		if time.Now().UTC().After(tokenRecord.ExpiresAt) {
			rw.WriteHeader(401)
			return
		}

		queryParams := database.RevokeRefreshTokenParams{
			Token: token,
			RevokedAt: sql.NullTime{
				Valid: true,
				Time:  time.Now().UTC(),
			},
		}
		_, revokeTokenErr := cfg.dbQueries.RevokeRefreshToken(req.Context(), queryParams)
		if revokeTokenErr != nil {
			resp := errorReturnVals{
				Error: revokeTokenErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(401)
			rw.Write(data)
			return
		}

		rw.WriteHeader(204)
	})

	mux.HandleFunc("POST /api/users", func(rw http.ResponseWriter, req *http.Request) {
		type parameters struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		type errorReturnVals struct {
			Error string `json:"error"`
		}

		type userReturnVals struct {
			ID          uuid.UUID `json:"id"`
			CreatedAt   time.Time `json:"created_at"`
			UpdatedAt   time.Time `json:"updated_at"`
			Email       string    `json:"email"`
			IsChirpyRed bool      `json:"is_chirpy_red"`
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

		hash, hashingErr := auth.HashPassword(params.Password)

		if hashingErr != nil {
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

		queryParams := database.CreateUserParams{
			Email:          params.Email,
			HashedPassword: hash,
		}

		user, dbErr := cfg.dbQueries.CreateUser(req.Context(), queryParams)

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
			ID:          user.ID,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			Email:       user.Email,
			IsChirpyRed: user.IsChirpyRed,
		}

		data, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			rw.WriteHeader(500)
			return
		}

		rw.WriteHeader(201)
		rw.Write(data)
	})

	mux.HandleFunc("PUT /api/users", func(rw http.ResponseWriter, req *http.Request) {
		type parameters struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		type errorReturnVals struct {
			Error string `json:"error"`
		}

		type userReturnVals struct {
			ID          uuid.UUID `json:"id"`
			CreatedAt   time.Time `json:"created_at"`
			UpdatedAt   time.Time `json:"updated_at"`
			Email       string    `json:"email"`
			IsChirpyRed bool      `json:"is_chirpy_red"`
		}

		rw.Header().Set("Content-Type", "application/json")

		token, tokenErr := auth.GetBearerToken(req.Header)
		if tokenErr != nil {
			resp := errorReturnVals{
				Error: tokenErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(401)
			rw.Write(data)
			return
		}

		userId, validateErr := auth.ValidateJWT(token, cfg.jwtSecret)
		if validateErr != nil {
			resp := errorReturnVals{
				Error: validateErr.Error(),
			}

			data, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				rw.WriteHeader(500)
				return
			}

			rw.WriteHeader(401)
			rw.Write(data)
			return
		}

		decoder := json.NewDecoder(req.Body)
		params := parameters{}
		parseErr := decoder.Decode(&params)
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

		hash, hashingErr := auth.HashPassword(params.Password)
		if hashingErr != nil {
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

		queryParams := database.UpdateUserParams{
			Email:          params.Email,
			HashedPassword: hash,
			UpdatedAt:      time.Now().UTC(),
			ID:             userId,
		}

		updatedUser, dbErr := cfg.dbQueries.UpdateUser(req.Context(), queryParams)
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
			ID:        userId,
			CreatedAt: updatedUser.CreatedAt,
			UpdatedAt: updatedUser.UpdatedAt,
			Email:     updatedUser.Email,
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
