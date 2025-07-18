package auth

import (
	"errors"
	"fmt"
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

	exp, _ := token.Claims.GetExpirationTime()

	fmt.Printf("Now: %v\nExp: %v", time.Now().UTC(), exp)

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
