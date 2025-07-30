package routes

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/auth"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/config"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/database"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/models"
)

func MapAuthRoutes(mux *http.ServeMux, cfg *config.AppConfig) {
	basePath := "/api"

	MapPost(mux, fmt.Sprintf("%s/login", basePath), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		params, parseErr := parseParams[models.UserCredentialsParameters](r)
		if parseErr != nil {
			status, body := returnJsonError(parseErr, http.StatusBadRequest)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		user, dbErr := cfg.DbQueries.GetUserByEmail(r.Context(), params.Email)
		if dbErr != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("dbErr: Incorrect email or password"))
			return
		}

		if params.Email != user.Email {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("user email does not match: incorrect email or password"))
			return
		}

		hasingErr := auth.CheckPasswordHash(params.Password, user.HashedPassword)
		if hasingErr != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Check Password Hash: Incorrect email or password"))
			return
		}

		token, tokenErr := auth.MakeJWT(user.ID, cfg.JwtSecret, 60*time.Second)
		if tokenErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Could not create JWT."))
			return
		}

		refreshToken, refreshTokenErr := auth.MakeRefreshToken()
		if refreshTokenErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Could not create Refresh Token."))
			return
		}

		refreshExpiration := time.Now().UTC().Add(time.Hour * 24 * 60)

		queryParams := database.SaveTokenParams{
			Token:     refreshToken,
			UserID:    user.ID,
			ExpiresAt: refreshExpiration,
		}
		_, dbRefreshTokenErr := cfg.DbQueries.SaveToken(r.Context(), queryParams)
		if dbRefreshTokenErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Could not create Refresh Token."))
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
		returnJsonResponse(resp, http.StatusOK)
	})

	MapPost(mux, fmt.Sprintf("%s/refresh", basePath), func(rw http.ResponseWriter, req *http.Request) {
		token, tokenErr := auth.GetBearerToken(req.Header)
		if tokenErr != nil {
			status, body := returnJsonError(tokenErr, http.StatusUnauthorized)
			rw.Write(body)
			rw.WriteHeader(status)
			return
		}

		tokenRecord, getDbTokenErr := cfg.DbQueries.GetUserFromRefreshToken(req.Context(), token)
		if getDbTokenErr != nil {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		if tokenRecord.RevokedAt.Valid && time.Now().UTC().After(tokenRecord.RevokedAt.Time) {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		if time.Now().UTC().After(tokenRecord.ExpiresAt) {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		bearer, bearerErr := auth.MakeJWT(tokenRecord.UserID, cfg.JwtSecret, time.Hour)
		if bearerErr != nil {
			rw.Write([]byte("Could not create bearer token"))
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp := models.RefreshTokenResponse{
			Token: bearer,
		}
		returnJsonResponse(resp, http.StatusOK)
	})

	MapPost(mux, fmt.Sprintf("%s/revoke", basePath), func(w http.ResponseWriter, r *http.Request) {
		token, tokenErr := auth.GetBearerToken(r.Header)
		if tokenErr != nil {
			status, body := returnJsonError(tokenErr, http.StatusUnauthorized)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		tokenRecord, getDbTokenErr := cfg.DbQueries.GetUserFromRefreshToken(r.Context(), token)
		if getDbTokenErr != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if tokenRecord.RevokedAt.Valid && time.Now().UTC().After(tokenRecord.RevokedAt.Time) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if time.Now().UTC().After(tokenRecord.ExpiresAt) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		queryParams := database.RevokeRefreshTokenParams{
			Token: token,
			RevokedAt: sql.NullTime{
				Valid: true,
				Time:  time.Now().UTC(),
			},
		}
		_, revokeTokenErr := cfg.DbQueries.RevokeRefreshToken(r.Context(), queryParams)
		if revokeTokenErr != nil {
			status, body := returnJsonError(revokeTokenErr, http.StatusUnauthorized)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}
