package password

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// HashPassword hashes the password using Argon2
func HashPassword(pass string) string {
	salt := make([]byte, 16)
	rand.Read(salt)
	h := argon2.Key([]byte(pass), salt, 1, 2*1024*1024, 4, 32)
	// Encode both parts; separator can stay as ':'.
	return fmt.Sprintf("%x:%x", h, salt)
}

func CheckPassword(pass, hash string) bool {
    parts := strings.Split(hash, ":")
    if len(parts) != 2 {
        return false
    }
    hashHex, saltHex := parts[0], parts[1]
    saltBytes, err := hex.DecodeString(saltHex)
    if err != nil {
        return false
    }
    // Re‑calculate the hash using the original password and decoded salt.
    r := argon2.Key([]byte(pass), saltBytes, 1, 2*1024*1024, 4, 32)
    // Compare the hex‑encoded hash.
    return hex.EncodeToString(r) == hashHex
}
