package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	bytes, cryptErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), cryptErr
}

func CheckPasswordHash(password, hash string) error {
	//error: Use the bcrypt.CompareHashAndPassword function to compare the password
	// that the user entered in the HTTP request with the password that is stored in the database.
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
