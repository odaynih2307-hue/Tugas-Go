package helper

import "golang.org/x/crypto/bcrypt"

const PasswordHashCost = 12

// HashPassword mengubah password asli menjadi bcrypt hash.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		PasswordHashCost,
	)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// CheckPassword membandingkan password asli dengan bcrypt hash.
func CheckPassword(password, passwordHash string) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password),
	)

	return err == nil
}
