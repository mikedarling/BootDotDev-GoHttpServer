package routes

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/auth"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/config"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/database"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/models"
)

func MapUserRoutes(mux *http.ServeMux, cfg *config.AppConfig) {
	basePath := "/api/users"

	MapPost(mux, basePath, func(rw http.ResponseWriter, req *http.Request) {
		decoder := json.NewDecoder(req.Body)
		params := models.UserCredentialsParameters{}
		parseErr := decoder.Decode(&params)

		rw.Header().Set("Content-Type", "application/json")

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

		hash, hashingErr := auth.HashPassword(params.Password)

		if hashingErr != nil {
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

		queryParams := database.CreateUserParams{
			Email:          params.Email,
			HashedPassword: hash,
		}

		user, dbErr := cfg.DbQueries.CreateUser(req.Context(), queryParams)

		if dbErr != nil {
			resp := models.ErrorResponse{
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

		resp := models.UserResponse{
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

	MapPut(mux, basePath, func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")

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

		userId, validateErr := auth.ValidateJWT(token, cfg.JwtSecret)
		if validateErr != nil {
			resp := models.ErrorResponse{
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

		hash, hashingErr := auth.HashPassword(params.Password)
		if hashingErr != nil {
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

		queryParams := database.UpdateUserParams{
			Email:          params.Email,
			HashedPassword: hash,
			UpdatedAt:      time.Now().UTC(),
			ID:             userId,
		}

		updatedUser, dbErr := cfg.DbQueries.UpdateUser(req.Context(), queryParams)
		if dbErr != nil {
			resp := models.ErrorResponse{
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

		resp := models.UserResponse{
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

	MapPost(mux, "/api/polka/webhooks", func(rw http.ResponseWriter, req *http.Request) {

		rw.Header().Set("Content-Type", "application/json")

		apiKey, apiKeyErr := auth.GetAPIKey(req.Header)
		if apiKeyErr != nil {
			resp := models.ErrorResponse{
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
		params := models.PolkaWebhookParamters{}
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

		if params.Event != "user.upgraded" {
			rw.WriteHeader(204)
			return
		}

		_, dbErr := cfg.DbQueries.UpgradeUserById(req.Context(), params.Data.UserId)
		if dbErr != nil {
			resp := models.ErrorResponse{
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
}
