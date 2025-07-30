package routes

import (
	"net/http"
	"time"

	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/auth"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/config"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/database"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/models"
)

func MapUserRoutes(mux *http.ServeMux, cfg *config.AppConfig) {
	basePath := "/api/users"

	MapPost(mux, basePath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		params, parseErr := parseParams[models.UserCredentialsParameters](r)
		if parseErr != nil {
			status, body := returnJsonError(parseErr, http.StatusBadRequest)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		hash, hashingErr := auth.HashPassword(params.Password)
		if hashingErr != nil {
			status, body := returnJsonError(hashingErr, http.StatusBadRequest)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		queryParams := database.CreateUserParams{
			Email:          params.Email,
			HashedPassword: hash,
		}
		user, dbErr := cfg.DbQueries.CreateUser(r.Context(), queryParams)
		if dbErr != nil {
			status, body := returnJsonError(dbErr, http.StatusBadRequest)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		resp := models.UserResponse{
			ID:          user.ID,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			Email:       user.Email,
			IsChirpyRed: user.IsChirpyRed,
		}
		returnJsonResponse(resp, http.StatusCreated)
	})

	MapPut(mux, basePath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		token, tokenErr := auth.GetBearerToken(r.Header)
		if tokenErr != nil {
			status, body := returnJsonError(tokenErr, http.StatusUnauthorized)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		userId, validateErr := auth.ValidateJWT(token, cfg.JwtSecret)
		if validateErr != nil {
			status, body := returnJsonError(validateErr, http.StatusUnauthorized)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		params, parseErr := parseParams[models.UserCredentialsParameters](r)
		if parseErr != nil {
			status, body := returnJsonError(parseErr, http.StatusBadRequest)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		hash, hashingErr := auth.HashPassword(params.Password)
		if hashingErr != nil {
			status, body := returnJsonError(hashingErr, http.StatusBadRequest)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		queryParams := database.UpdateUserParams{
			Email:          params.Email,
			HashedPassword: hash,
			UpdatedAt:      time.Now().UTC(),
			ID:             userId,
		}
		updatedUser, dbErr := cfg.DbQueries.UpdateUser(r.Context(), queryParams)
		if dbErr != nil {
			status, body := returnJsonError(dbErr, http.StatusBadRequest)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		resp := models.UserResponse{
			ID:        userId,
			CreatedAt: updatedUser.CreatedAt,
			UpdatedAt: updatedUser.UpdatedAt,
			Email:     updatedUser.Email,
		}
		returnJsonResponse(resp, http.StatusOK)
	})

	MapPost(mux, "/api/polka/webhooks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		apiKey, apiKeyErr := auth.GetAPIKey(r.Header)
		if apiKeyErr != nil {
			status, body := returnJsonError(apiKeyErr, http.StatusUnauthorized)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		if apiKey != cfg.PolkaKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		params, parseParamsErr := parseParams[models.PolkaWebhookParamters](r)
		if parseParamsErr != nil {
			status, body := returnJsonError(parseParamsErr, http.StatusBadRequest)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		if params.Event != "user.upgraded" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		_, dbErr := cfg.DbQueries.UpgradeUserById(r.Context(), params.Data.UserId)
		if dbErr != nil {
			status, body := returnJsonError(dbErr, http.StatusNotFound)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}
