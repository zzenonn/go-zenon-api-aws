package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zzenonn/go-zenon-api-aws/internal/domain"
	"github.com/zzenonn/go-zenon-api-aws/internal/errors"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository - interface for data access methods
type UserRepository interface {
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	GetUser(ctx context.Context, username string) (domain.User, error)
	UpdateUser(ctx context.Context, id string, user domain.User) (domain.User, error)
	DeleteUser(ctx context.Context, id string) error
	GetAllUsers(ctx context.Context, pageSize int, nextToken string) ([]domain.User, string, error)
}

// UserService - service for managing users
type UserService struct {
	Repo UserRepository
}

// NewUserService - returns a new instance of UserService
func NewUserService(repo UserRepository) *UserService {
	return &UserService{
		Repo: repo,
	}
}

func (s *UserService) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	// Check if the username already exists
	existingUser, err := s.Repo.GetUser(ctx, *user.Username)
	if err == nil && existingUser.Id != "" {
		// If a user with the username already exists, return an error
		return domain.User{}, errors.ErrInvalidUser
	}

	// Proceed if the username doesn't exist
	user.Id = uuid.New().String()

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

	user, err = s.Repo.UpdateUser(ctx, userToUpdate.Id, user)

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

	err = s.Repo.DeleteUser(ctx, userToDelete.Id)

	return err
}

// TODO send email vefification
func (s *UserService) Signup(ctx context.Context, user domain.User) (domain.User, error) {
	user.Id = uuid.New().String()

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
