package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeJWT_DoesNotError(t *testing.T) {
	// Arrange
	userId := uuid.New()
	secret := "MyT0k3n5ecr37"
	duration := time.Hour * 24

	// Act
	_, createErr := MakeJWT(userId, secret, time.Duration(duration))

	// Assert
	if createErr != nil {
		t.Errorf("Error creating token:\n\t%v", createErr)
	}
}

func TestValidateJWT_GoodToken_ReturnsUserId(t *testing.T) {
	// Arrange
	userId := uuid.New()
	secret := "MyT0k3n5ecr37"
	duration := time.Hour * 24
	token, createErr := MakeJWT(userId, secret, time.Duration(duration))

	if createErr != nil {
		t.FailNow()
	}

	// Act
	returnedID, validateErr := ValidateJWT(token, secret)

	// Assert
	if validateErr != nil {
		t.Errorf("\nCould not validate token:\n  %v", validateErr)
	}
	if returnedID != userId {
		t.Errorf("\nWrong ID returned.\n\tExpected: %v\n  Actual: %v", userId, returnedID)
	}
}

func TestValidateJWT_ExpiredToken_IsInvalid(t *testing.T) {
	// Arrange
	userId := uuid.New()
	secret := "MyT0k3n5ecr37"
	duration := time.Hour * -24
	token, createErr := MakeJWT(userId, secret, time.Duration(duration))

	if createErr != nil {
		t.FailNow()
	}

	// Act
	_, validateErr := ValidateJWT(token, secret)

	// Assert
	if validateErr == nil {
		t.Error("\nToken should not be valid.")
	}
}

func TestValidateJWT_WrongSecret_IsInvalid(t *testing.T) {
	// Arrange
	userId := uuid.New()
	secret := "MyT0k3n5ecr37"
	duration := time.Hour * 24
	token, createErr := MakeJWT(userId, secret, time.Duration(duration))

	if createErr != nil {
		t.FailNow()
	}

	// Act
	_, validateErr := ValidateJWT(token, "badsecret")

	// Assert
	if validateErr == nil {
		t.Error("\nToken should not be valid.")
	}
}

func TestGetBearerToken_GoodHeader_ReturnsToken(t *testing.T) {
	// Arrange
	headers := make(map[string][]string)
	headers["Authorization"] = make([]string, 1)
	headers["Authorization"][0] = "Bearer thisIsMyToken"

	// Act
	token, err := GetBearerToken(headers)

	// Assert
	if err != nil {
		t.Errorf("Error should be nil, but was %v", err)
	}
	if token == "" {
		t.Error("Empty string was returned when expecting a token.")
	}
}

func TestGetBearerToken_NoAuthHeader_ReturnsError(t *testing.T) {
	// Arrange
	headers := make(map[string][]string)

	// Act
	_, err := GetBearerToken(headers)

	// Assert
	if err == nil {
		t.Error("Error should not be nil")
	}
}

func TestGetBearerToken_EmptyAuthHeader_ReturnsError(t *testing.T) {
	// Arrange
	headers := make(map[string][]string)
	headers["Authorization"] = make([]string, 1)
	headers["Authorization"][0] = ""

	// Act
	_, err := GetBearerToken(headers)

	// Assert
	if err == nil {
		t.Error("Error should not be nil")
	}
}

func TestGetBearerToken_EmptyTokenString_ReturnsError(t *testing.T) {
	// Arrange
	headers := make(map[string][]string)
	headers["Authorization"] = make([]string, 1)
	headers["Authorization"][0] = "Bearer"

	// Act
	_, err := GetBearerToken(headers)

	// Assert
	if err == nil {
		t.Error("Error should not be nil")
	}
}
