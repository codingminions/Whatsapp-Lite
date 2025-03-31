package token

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Errors
var (
	ErrInvalidToken = errors.New("token is invalid")
	ErrExpiredToken = errors.New("token has expired")
)

// ValidationError is returned when token validation fails
type ValidationError struct {
	Err error
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("token validation failed: %v", e.Err)
}

// Payload contains the payload data of the token
type Payload struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}

// Maker is an interface for managing tokens
type Maker interface {
	// CreateToken creates a new token for a specific user
	CreateToken(userID, username string, duration time.Duration) (string, *Payload, error)

	// VerifyToken checks if the token is valid
	VerifyToken(token string) (*Payload, error)
}

// JWTMaker is a JSON Web Token maker
type JWTMaker struct {
	secretKey string
}

// NewJWTMaker creates a new JWTMaker
func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < 32 {
		return nil, errors.New("secret key must be at least 32 characters")
	}
	return &JWTMaker{secretKey: secretKey}, nil
}

// CreateToken creates a new token for a specific user
func (maker *JWTMaker) CreateToken(userID, username string, duration time.Duration) (string, *Payload, error) {
	payload := &Payload{
		UserID:    userID,
		Username:  username,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(duration),
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":    payload.UserID,
		"username":   payload.Username,
		"issued_at":  payload.IssuedAt.Unix(),
		"expired_at": payload.ExpiredAt.Unix(),
	})

	tokenString, err := jwtToken.SignedString([]byte(maker.secretKey))
	if err != nil {
		return "", nil, err
	}

	return tokenString, payload, nil
}

// VerifyToken checks if the token is valid
func (maker *JWTMaker) VerifyToken(token string) (*Payload, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, ValidationError{Err: fmt.Errorf("unexpected signing method: %v", token.Header["alg"])}
		}
		return []byte(maker.secretKey), nil
	}

	jwtToken, err := jwt.Parse(token, keyFunc)
	if err != nil {
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			return nil, ValidationError{Err: ErrInvalidToken}
		}
		return nil, ValidationError{Err: err}
	}

	if !jwtToken.Valid {
		return nil, ValidationError{Err: ErrInvalidToken}
	}

	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ValidationError{Err: ErrInvalidToken}
	}

	// Extract claims
	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, ValidationError{Err: ErrInvalidToken}
	}

	username, ok := claims["username"].(string)
	if !ok {
		return nil, ValidationError{Err: ErrInvalidToken}
	}

	issuedAtFloat, ok := claims["issued_at"].(float64)
	if !ok {
		return nil, ValidationError{Err: ErrInvalidToken}
	}

	expiredAtFloat, ok := claims["expired_at"].(float64)
	if !ok {
		return nil, ValidationError{Err: ErrInvalidToken}
	}

	issuedAt := time.Unix(int64(issuedAtFloat), 0)
	expiredAt := time.Unix(int64(expiredAtFloat), 0)

	// Check if the token has expired
	if time.Now().After(expiredAt) {
		return nil, ValidationError{Err: ErrExpiredToken}
	}

	payload := &Payload{
		UserID:    userID,
		Username:  username,
		IssuedAt:  issuedAt,
		ExpiredAt: expiredAt,
	}

	return payload, nil
}

// GenerateRandomString generates a random string of the specified length
func GenerateRandomString(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:length], nil
}
