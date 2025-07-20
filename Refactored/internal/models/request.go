package models

import "github.com/google/uuid"

type UserCredentialsParameters struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ChirpsParameters struct {
	Body string `json:"body"`
}

type PolkaWebhookParamters struct {
	Event string `json:"event"`
	Data  struct {
		UserId uuid.UUID `json:"user_id"`
	} `json:"data"`
}
