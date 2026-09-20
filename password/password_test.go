package password

import(
	"testing"
)

func TestHashPass(t *testing.T) {
	pass := "plain-pass"
	hash := HashPassword(pass)

	if hash == pass {
		t.Errorf("password should be different after hashing, got %v", hash)
	}
	tt := []struct{
		want string
		pass string
	}
}

func TestCheckPassword(t *testing.T) {
	pass := "plain-pass"
	hash := HashPassword(pass)
	if !CheckPassword(pass, hash) {
		t.Errorf("password should match the hash, got %v", hash)
	}
}
