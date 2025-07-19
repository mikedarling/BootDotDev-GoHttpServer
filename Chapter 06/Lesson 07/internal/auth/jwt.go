package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	now := time.Now().UTC()

	issuedAt := jwt.NumericDate{
		Time: now,
	}

	expiresAt := jwt.NumericDate{
		Time: now.Add(expiresIn),
	}

	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  &issuedAt,
		ExpiresAt: &expiresAt,
		Subject:   userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {

	claims := jwt.RegisteredClaims{}

	token, parseErr := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {

		return []byte(tokenSecret), nil
	})

	if parseErr != nil {
		return uuid.Nil, parseErr
	}

	if !token.Valid {
		return uuid.Nil, errors.New("token is invalid")
	}

	subj, claimSubErr := claims.GetSubject()
	if claimSubErr != nil {
		return uuid.Nil, errors.New("could not get claim subject")
	}

	id, uuidParseErr := uuid.Parse(subj)
	if uuidParseErr != nil {
		return uuid.Nil, errors.New("could not parse claim subject string to UUID")
	}

	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	bearer := headers.Get("Authorization")
	if bearer == "" {
		return "", errors.New("No Authorization header available")
	}

	bearer = strings.TrimSpace(strings.ReplaceAll(bearer, "Bearer", ""))
	if bearer == "" {
		return "", errors.New("Bearer token must not be empty")
	}

	return bearer, nil
}
