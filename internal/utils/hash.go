package utils

import "golang.org/x/crypto/bcrypt"

// Generate a dummy hash to help with security so we can compare it password
func GenerateDummyHash() []byte {
	dummyHash, _ := bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.DefaultCost)
	return dummyHash
}
