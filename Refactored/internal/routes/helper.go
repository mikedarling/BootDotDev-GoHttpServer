package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/models"
)

func MapGet(this *http.ServeMux, route string, handler func(w http.ResponseWriter, r *http.Request)) {
	pattern := fmt.Sprintf("GET %s", route)
	(*this).HandleFunc(pattern, handler)
}

func MapPost(this *http.ServeMux, route string, handler func(w http.ResponseWriter, r *http.Request)) {
	pattern := fmt.Sprintf("POST %s", route)
	(*this).HandleFunc(pattern, handler)
}

func MapPut(this *http.ServeMux, route string, handler func(w http.ResponseWriter, r *http.Request)) {
	pattern := fmt.Sprintf("PUT %s", route)
	(*this).HandleFunc(pattern, handler)
}

func MapDelete(this *http.ServeMux, route string, handler func(w http.ResponseWriter, r *http.Request)) {
	pattern := fmt.Sprintf("DELETE %s", route)
	(*this).HandleFunc(pattern, handler)
}

func returnJsonError(err error, responseCode int) (int, []byte) {
	resp := models.ErrorResponse{
		Error: err.Error(),
	}

	data, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		return http.StatusInternalServerError, nil
	}

	return responseCode, data
}

func returnJsonResponse[T any](resp T, responseCode int) (int, []byte) {
	data, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		return http.StatusInternalServerError, nil
	}

	return responseCode, []byte(data)
}

func parseParams[T any](r *http.Request) (model T, err error) {
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&model)
	return
}
