package gateman

import (
	"crypto/rand"
	"golang.org/x/crypto/argon2"
)

// HashPass hashes the password using Argon2
func HashPass(pass string) string {
	salt := make([]byte, 16)
	rand.Read(salt)
	h := argon2.IDKey([]byte(pass), salt, 1, 64*1024, 4, 32)
	return string(h)
}

// CheckPass checks if password matches the hash
func CheckPass(pass, hash string) bool {
	return HashPass(pass) == hash
}
