package service

import (
	"context"
	"io"

	"github.com/zzenonn/go-zenon-api-aws/internal/domain"
	"github.com/zzenonn/go-zenon-api-aws/internal/errors"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository - interface for data access methods
type UserRepository interface {
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	GetUser(ctx context.Context, username string) (domain.User, error)
	UpdateUser(ctx context.Context, username string, user domain.User) (domain.User, error)
	DeleteUser(ctx context.Context, username string) error
	GetAllUsers(ctx context.Context, pageSize int, nextToken string) ([]domain.User, string, error)
}

// UserProfileRepository - interface for profile storage operations
type UserProfileRepository interface {
	Upload(ctx context.Context, key string, r io.Reader) error
	GetPresignedUrl(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}

// UserService - service for managing users and profiles
type UserService struct {
	Repo        UserRepository
	ProfileRepo UserProfileRepository
}

// NewUserService - returns a new instance of UserService
func NewUserService(repo UserRepository, profileRepo UserProfileRepository) *UserService {
	return &UserService{
		Repo:        repo,
		ProfileRepo: profileRepo,
	}
}

func (s *UserService) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	// Check if the username already exists
	existingUser, err := s.Repo.GetUser(ctx, *user.Username)
	if err == nil && existingUser.Username != nil {
		// If a user with the username already exists, return an error
		return domain.User{}, errors.ErrInvalidUser
	}

	if err := user.HashPassword(); err != nil {
		return domain.User{}, err
	}

	insertedUser, err := s.Repo.CreateUser(ctx, user)
	if err != nil {
		return domain.User{}, err
	}

	return insertedUser, nil
}

func (s *UserService) GetUser(ctx context.Context, username string) (domain.User, error) {

	user, err := s.Repo.GetUser(ctx, username)

	if err != nil {
		return domain.User{}, errors.ErrInvalidUser
	}

	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, user domain.User) (domain.User, error) {

	userToUpdate, err := s.Repo.GetUser(ctx, *user.Username)

	if err != nil {
		return domain.User{}, err
	}

	// if the password is not empty, hash it
	if user.Password != "" {
		if err := user.HashPassword(); err != nil {
			return domain.User{}, err
		}
	}

	user, err = s.Repo.UpdateUser(ctx, *userToUpdate.Username, user)

	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, username string) error {

	userToDelete, err := s.Repo.GetUser(ctx, username)

	if err != nil {
		return err
	}

	err = s.Repo.DeleteUser(ctx, *userToDelete.Username)

	return err
}

// TODO send email vefification
func (s *UserService) Signup(ctx context.Context, user domain.User) (domain.User, error) {

	if err := user.HashPassword(); err != nil {
		return domain.User{}, err
	}

	insertedUser, err := s.Repo.CreateUser(ctx, user)
	if err != nil {
		return domain.User{}, err
	}

	return insertedUser, nil
}

func (s *UserService) Login(ctx context.Context, username string, password string) error {
	user, err := s.Repo.GetUser(ctx, username)
	if err != nil {
		return err
	}

	if bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(password)) != nil {
		return errors.ErrInvalidUser
	}

	return nil
}

// UploadProfile uploads a user profile image
func (s *UserService) UploadProfile(ctx context.Context, username string, key string, r io.Reader) error {
	profilePath := username + "/profile/" + key
	err := s.ProfileRepo.Upload(ctx, profilePath, r)
	if err != nil {
		return err
	}

	// Update user's profile path in database
	user, err := s.Repo.GetUser(ctx, username)
	if err != nil {
		return err
	}

	user.ProfilePath = &profilePath
	_, err = s.Repo.UpdateUser(ctx, username, user)
	return err
}

// GeneratePresignedURL generates a pre-signed URL for accessing a profile image
func (s *UserService) GeneratePresignedURL(ctx context.Context, username string, key string) (string, error) {
	return s.ProfileRepo.GetPresignedUrl(ctx, username+"/profile/"+key)
}

// DeleteProfile deletes a user profile image
func (s *UserService) DeleteProfile(ctx context.Context, username string, key string) error {
	profilePath := username + "/profile/" + key
	err := s.ProfileRepo.Delete(ctx, profilePath)
	if err != nil {
		return err
	}

	// Clear user's profile path in database
	user, err := s.Repo.GetUser(ctx, username)
	if err != nil {
		return err
	}

	user.ProfilePath = nil
	_, err = s.Repo.UpdateUser(ctx, username, user)
	return err
}
