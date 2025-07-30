package routes

import (
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/auth"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/config"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/database"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/models"
)

func MapChirpRoutes(mux *http.ServeMux, cfg *config.AppConfig) {
	basePath := "/api/chirps"

	MapGet(mux, basePath, func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		authorId := queryParams.Get("author_id")

		var userId uuid.UUID
		var userIdParseErr error

		if authorId != "" {
			userId, userIdParseErr = uuid.Parse(authorId)
			if userIdParseErr != nil {
				status, body := returnJsonError(userIdParseErr, http.StatusBadRequest)
				w.WriteHeader(status)
				w.Write(body)
				return
			}
		}

		var chirps []database.Chirp
		var dbErr error

		if uuid.Nil != userId {
			chirps, dbErr = cfg.DbQueries.GetChirpsByUser(r.Context(), userId)
		} else {
			chirps, dbErr = cfg.DbQueries.GetAllChirps(r.Context())
		}

		sortOrder := queryParams.Get("sort")
		if sortOrder != "" && sortOrder == "desc" {
			sort.Slice(chirps, func(i, j int) bool { return chirps[j].CreatedAt.Before(chirps[i].CreatedAt) })
		}

		if dbErr != nil {
			status, body := returnJsonError(dbErr, http.StatusBadRequest)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		resp := make([]models.ChirpResponse, len(chirps))

		for i, chirp := range chirps {
			resp[i] = models.ChirpResponse{
				ID:        chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserId:    chirp.UserID,
			}
		}

		returnJsonResponse(resp, http.StatusOK)
	})

	MapGet(mux, fmt.Sprintf("%s/{chirpID}", basePath), func(w http.ResponseWriter, r *http.Request) {
		chirpId, uuidParseErr := uuid.Parse(r.PathValue("chirpID"))
		if uuidParseErr != nil {
			status, body := returnJsonError(uuidParseErr, http.StatusBadRequest)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		chirp, dbErr := cfg.DbQueries.GetChirp(r.Context(), chirpId)
		if dbErr != nil {
			status, body := returnJsonError(dbErr, http.StatusNotFound)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		resp := models.ChirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserId:    chirp.UserID,
		}
		returnJsonResponse(resp, http.StatusOK)
	})

	MapDelete(mux, fmt.Sprintf("%s/{chirpID}", basePath), func(w http.ResponseWriter, r *http.Request) {
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

		chirpId, uuidParseErr := uuid.Parse(r.PathValue("chirpID"))
		if uuidParseErr != nil {
			status, body := returnJsonError(uuidParseErr, http.StatusNotFound)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		chirp, chirpDbErr := cfg.DbQueries.GetChirpById(r.Context(), chirpId)
		if chirpDbErr != nil {
			status, body := returnJsonError(chirpDbErr, http.StatusUnauthorized)
			w.WriteHeader(status)
			w.Write(body)
		}

		if chirp.UserID != userId {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		delChirpErr := cfg.DbQueries.DeleteChirpById(r.Context(), chirp.ID)
		if delChirpErr != nil {
			status, body := returnJsonError(delChirpErr, http.StatusUnauthorized)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	MapPost(mux, basePath, func(w http.ResponseWriter, r *http.Request) {
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

		params, parseParamsErr := parseParams[models.ChirpsParameters](r)
		if parseParamsErr != nil {
			status, body := returnJsonError(parseParamsErr, http.StatusBadRequest)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		valid := len(params.Body) < 141
		if !valid {
			status, body := returnJsonError(tokenErr, http.StatusBadRequest)
			w.WriteHeader(status)
			w.Write(body)
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

		chirp, dbErr := cfg.DbQueries.CreateChirps(r.Context(), queryParams)
		if dbErr != nil {
			status, body := returnJsonError(dbErr, http.StatusBadRequest)
			w.WriteHeader(status)
			w.Write(body)
			return
		}

		resp := models.ChirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserId:    chirp.UserID,
		}
		returnJsonResponse(resp, http.StatusCreated)
	})
}
