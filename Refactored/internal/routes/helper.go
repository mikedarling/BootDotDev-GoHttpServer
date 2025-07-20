package routes

import (
	"fmt"
	"net/http"
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
