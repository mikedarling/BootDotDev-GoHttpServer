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
