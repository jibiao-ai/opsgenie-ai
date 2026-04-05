package service

import (
	"errors"
	"strings"
	"unicode"

	"github.com/jibiao-ai/cloud-agent/internal/model"
	"github.com/jibiao-ai/cloud-agent/internal/repository"
)

func GetUserByID(id uint, user *model.User) error {
	return repository.DB.First(user, id).Error
}

// isBcryptHash checks whether s is already a bcrypt hash (starts with $2a$, $2b$, etc.)
func isBcryptHash(s string) bool {
	return strings.HasPrefix(s, "$2a$") || strings.HasPrefix(s, "$2b$") || strings.HasPrefix(s, "$2y$")
}

// validatePassword enforces password strength policy:
// at least 9 chars, must contain uppercase, lowercase, digit, and special character.
func validatePassword(password string) error {
	if len(password) < 9 {
		return errors.New("密码长度至少9位")
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}
	if !hasUpper {
		return errors.New("密码必须包含大写字母")
	}
	if !hasLower {
		return errors.New("密码必须包含小写字母")
	}
	if !hasDigit {
		return errors.New("密码必须包含数字")
	}
	if !hasSpecial {
		return errors.New("密码必须包含特殊字符")
	}
	return nil
}

// GetUsers returns all users.
func GetUsers() ([]model.User, error) {
	var users []model.User
	if err := repository.DB.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// CreateUser hashes the password (if plaintext) and persists the user.
func CreateUser(user *model.User) error {
	if user.Username == "" {
		return errors.New("username is required")
	}
	if user.Password == "" {
		return errors.New("password is required")
	}
	// Validate password strength only for plaintext passwords
	if !isBcryptHash(user.Password) {
		if err := validatePassword(user.Password); err != nil {
			return err
		}
		hashed, err := HashPassword(user.Password)
		if err != nil {
			return err
		}
		user.Password = hashed
	}
	if user.Role == "" {
		user.Role = "user"
	}
	return repository.DB.Create(user).Error
}

// UpdateUser updates mutable fields; re-hashes the password when a new plaintext value is provided.
func UpdateUser(user *model.User) error {
	var existing model.User
	if err := repository.DB.First(&existing, user.ID).Error; err != nil {
		return errors.New("user not found")
	}

	// Only update password when explicitly provided and different from current hash
	if user.Password != "" && !isBcryptHash(user.Password) {
		if err := validatePassword(user.Password); err != nil {
			return err
		}
		hashed, err := HashPassword(user.Password)
		if err != nil {
			return err
		}
		existing.Password = hashed
	}

	// Update other editable fields
	if user.Username != "" {
		existing.Username = user.Username
	}
	if user.Email != "" {
		existing.Email = user.Email
	}
	if user.Role != "" {
		existing.Role = user.Role
	}
	if user.Avatar != "" {
		existing.Avatar = user.Avatar
	}

	return repository.DB.Save(&existing).Error
}

// DeleteUser soft-deletes a user by ID.
func DeleteUser(id uint) error {
	return repository.DB.Delete(&model.User{}, id).Error
}
