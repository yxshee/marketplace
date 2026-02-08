package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const minimumPasswordLength = 8

var ErrWeakPassword = errors.New("password does not meet minimum length")

func HashPassword(plain string) (string, error) {
	if len(plain) < minimumPasswordLength {
		return "", ErrWeakPassword
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
