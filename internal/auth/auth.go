package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed), err
}

func CheckPasswordHash(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err
}

func MakeJWT(userID uuid.UUID, tokenSecret string) (string, error) {
	expiry := 3600 * time.Second
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
		Subject:   userID.String(),
	})
	ss, err := claims.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return ss, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	parsedToken, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	if !parsedToken.Valid {
		return uuid.Nil, errors.New("token invalid")
	}
	userid, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}
	return userid, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	tokenraw := headers.Get("Authorization")
	if tokenraw == "" {
		return "", errors.New("token doesnt exist")
	}
	tokenArr := strings.Split(tokenraw, " ")
	return tokenArr[1], nil
}

func MakeRefreshToken() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", errors.New("could not generate random")
	}
	encodedStr := hex.EncodeToString(key)
	return encodedStr, nil
}

func GetAPIKey(headers http.Header) (string, error) {
	keyraw := headers.Get("Authorization")
	if keyraw == "" {
		return "", errors.New("APIkey does not exist")
	}
	keyArr := strings.Split(keyraw, " ")
	return keyArr[1], nil
}
