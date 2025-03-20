package auth

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	secret := "chirpy"
	tokenStr, err := MakeJWT(userID, secret)
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

	// Testing accuracy
	tokenStr, err := MakeJWT(userID, secret)
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

	// Testing Wrong Secret
	diffSecret := "wrong"
	diffTknStr, err := MakeJWT(userID, diffSecret)
	if err != nil {
		t.Fatalf("MakeJWT (different secret) returned an error: %v", err)
	}
	_, err = ValidateJWT(diffTknStr, secret)
	if err == nil {
		t.Error("Expected wrong secret but got no error")
	}
}

func TestGetBearerToken(t *testing.T) {
	headers := http.Header{
		"Authorization": {"Bearer TOKEN_STRING"},
		"Content-Type":  {"application/json"},
	}

	token, err := GetBearerToken(headers)
	if err != nil {
		t.Fatalf("Could not find token error: %v", err)
	}
	if token != "TOKEN_STRING" {
		t.Errorf("Expected TOKEN_STRING got: %v", token)
	}
}

func TestMakeRefreshToken(t *testing.T) {
	asd, err := MakeRefreshToken()
	if (err != nil) || (len(asd) != 64) {
		t.Errorf("Returned %v", asd)
	}
}

func TestGetAPIKey(t *testing.T) {
	headers := http.Header{
		"Authorization": {"ApiKey THE_KEY_HERE"},
		"Content-Type":  {"application/json"},
	}

	asd, err := GetAPIKey(headers)
	if (err != nil) || (asd != "THE_KEY_HERE") {
		t.Errorf("Got error %v", err)
	}

}
