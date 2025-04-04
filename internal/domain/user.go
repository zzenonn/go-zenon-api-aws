package domain

import "golang.org/x/crypto/bcrypt"

// User - representation of a user in the system
type User struct {
	Id             string  `json:"id" firestore:"id"`
	Username       *string `json:"username,omitempty" firestore:"username,omitempty"`
	Password       string  `json:"-" firestore:"-"`
	HashedPassword []byte  `json:"-" firestore:"hashed_password,omitempty"`
}

// HashPassword - hashes the user's password
func (u *User) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), 10)
	if err != nil {
		return err
	}
	u.HashedPassword = hashedPassword
	u.Password = "" // Clear plain text password after hashing
	return nil
}
