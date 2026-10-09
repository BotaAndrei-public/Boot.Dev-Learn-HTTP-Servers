package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "secret"

	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT error: %v", err)
	}

	parsedID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT error: %v", err)
	}

	if parsedID != userID {
		t.Fatalf("Expected user ID %v, got %v", parsedID, userID)
	}
}

func TestExpiredJWT(t *testing.T) {
	userID := uuid.New()
	secret := "secret"

	token, err := MakeJWT(userID, secret, -time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT error: %v", err)
	}

	_, err = ValidateJWT(token, secret)
	if err == nil {
		t.Errorf("Expected error for expired token, but got none")
	}
}
func TestWrongSecretJWT(t *testing.T) {
	userID := uuid.New()
	secret := "secret"
	wrongSecret := "WSecret"

	token, err := MakeJWT(userID, wrongSecret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT error: %v", err)
	}

	_, err = ValidateJWT(token, secret)
	if err == nil {
		t.Errorf("Expected error for wrong secret, but got none")

	}
}
