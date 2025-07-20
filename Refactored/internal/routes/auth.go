package routes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/auth"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/config"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/database"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/models"
)

func MapAuthRoutes(mux *http.ServeMux, cfg *config.AppConfig) {
	basePath := "/api"

	MapPost(mux, fmt.Sprintf("%s/login", basePath), func(rw http.ResponseWriter, req *http.Request) {

		rw.Header().Set("Content-Type", "application/json")

		decoder := json.NewDecoder(req.Body)
		params := models.UserCredentialsParameters{}
		parseErr := decoder.Decode(&params)
		if parseErr != nil {
			resp := models.ErrorResponse{
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

		user, dbErr := cfg.DbQueries.GetUserByEmail(req.Context(), params.Email)
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

		token, tokenErr := auth.MakeJWT(user.ID, cfg.JwtSecret, 60*time.Second)
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

		_, dbRefreshTokenErr := cfg.DbQueries.SaveToken(req.Context(), queryParams)
		if dbRefreshTokenErr != nil {
			rw.WriteHeader(500)
			rw.Write([]byte("Could not create Refresh Token."))
			return
		}

		resp := models.UserResponse{
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

	MapPost(mux, fmt.Sprintf("%s/refresh", basePath), func(rw http.ResponseWriter, req *http.Request) {

		token, tokenErr := auth.GetBearerToken(req.Header)
		if tokenErr != nil {
			resp := models.ErrorResponse{
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

		tokenRecord, getDbTokenErr := cfg.DbQueries.GetUserFromRefreshToken(req.Context(), token)
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

		resp := models.RefreshTokenResponse{
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

	MapPost(mux, fmt.Sprintf("%s/revoke", basePath), func(rw http.ResponseWriter, req *http.Request) {
		token, tokenErr := auth.GetBearerToken(req.Header)
		if tokenErr != nil {
			resp := models.ErrorResponse{
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

		tokenRecord, getDbTokenErr := cfg.DbQueries.GetUserFromRefreshToken(req.Context(), token)
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
		_, revokeTokenErr := cfg.DbQueries.RevokeRefreshToken(req.Context(), queryParams)
		if revokeTokenErr != nil {
			resp := models.ErrorResponse{
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
}
