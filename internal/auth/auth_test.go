package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	secret := "chirpy"
	expiresIn := time.Hour
	tokenStr, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned an error: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("Expected a non-empty token string")
	}
	t.Log(tokenStr)
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "chirpy"
	expiresIn := time.Hour

	// Testing accuracy
	tokenStr, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned an error: %v", err)
	}
	jwtUserid, err := ValidateJWT(tokenStr, secret)
	if err != nil {
		t.Fatalf("ValidateJWT returned an error: %v", err)
	}
	if jwtUserid != userID {
		t.Errorf("Expected %v got %v", userID, jwtUserid)
	}

	// Testing Expired Token
	expiredTknStr, err := MakeJWT(userID, secret, -time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT (expired token) returned an error: %v", err)
	}
	_, err = ValidateJWT(expiredTknStr, secret)
	if err == nil {
		t.Error("Expected expired token but got no error")
	}

	// Testing Wrong Secret
	diffSecret := "wrong"
	diffTknStr, err := MakeJWT(userID, diffSecret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT (different secret) returned an error: %v", err)
	}
	_, err = ValidateJWT(diffTknStr, secret)
	if err == nil {
		t.Error("Expected wrong secret but got no error")
	}
}
