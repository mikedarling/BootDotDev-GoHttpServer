package routes

import (
	"encoding/json"
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

	MapGet(mux, basePath, func(rw http.ResponseWriter, req *http.Request) {
		queryParams := req.URL.Query()
		authorId := queryParams.Get("author_id")

		var userId uuid.UUID
		var userIdParseErr error
		if authorId != "" {
			userId, userIdParseErr = uuid.Parse(authorId)
			if userIdParseErr != nil {
				resp := models.ErrorResponse{
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
			chirps, dbErr = cfg.DbQueries.GetChirpsByUser(req.Context(), userId)
		} else {
			chirps, dbErr = cfg.DbQueries.GetAllChirps(req.Context())
		}

		sortOrder := queryParams.Get("sort")
		if sortOrder != "" && sortOrder == "desc" {
			sort.Slice(chirps, func(i, j int) bool { return chirps[j].CreatedAt.Before(chirps[i].CreatedAt) })
		}

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

		data, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			rw.WriteHeader(500)
			return
		}

		rw.WriteHeader(200)
		rw.Write(data)
	})

	MapGet(mux, fmt.Sprintf("%s/{chirpID}", basePath), func(rw http.ResponseWriter, req *http.Request) {
		chirpId, uuidParseErr := uuid.Parse(req.PathValue("chirpID"))
		if uuidParseErr != nil {
			resp := models.ErrorResponse{
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

		chirp, dbErr := cfg.DbQueries.GetChirp(req.Context(), chirpId)
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

		resp := models.ChirpResponse{
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

	MapDelete(mux, fmt.Sprintf("%s/{chirpID}", basePath), func(rw http.ResponseWriter, req *http.Request) {
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

		chirpId, uuidParseErr := uuid.Parse(req.PathValue("chirpID"))
		if uuidParseErr != nil {
			resp := models.ErrorResponse{
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

		chirp, chirpDbErr := cfg.DbQueries.GetChirpById(req.Context(), chirpId)
		if chirpDbErr != nil {
			resp := models.ErrorResponse{
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

		delChirpErr := cfg.DbQueries.DeleteChirpById(req.Context(), chirp.ID)
		if delChirpErr != nil {
			resp := models.ErrorResponse{
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

	MapPost(mux, basePath, func(rw http.ResponseWriter, req *http.Request) {
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
		params := models.ChirpsParameters{}
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

		valid := len(params.Body) < 141
		if !valid {
			resp := models.ErrorResponse{
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

		chirp, dbErr := cfg.DbQueries.CreateChirps(req.Context(), queryParams)
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

		resp := models.ChirpResponse{
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
}
