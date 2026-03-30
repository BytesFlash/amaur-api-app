package password

import (
	"golang.org/x/crypto/bcrypt"
)

const DefaultCost = bcrypt.DefaultCost

func Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), DefaultCost)
	return string(b), err
}

func Verify(plain, hashed string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}
