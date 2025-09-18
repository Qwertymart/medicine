package utils

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
)

const (
	MinPasswordLength = 8
	MaxPasswordLength = 100
	BcryptCost        = 3
)

func HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLength {
		return "", errors.New("password too short")
	}
	if len(password) > MaxPasswordLength {
		return "", errors.New("password too long")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

func CheckPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
