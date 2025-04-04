package db

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"gitlab.com/zzenonn/go-zenon-api-aws/internal/domain"
	"google.golang.org/api/iterator"
)

// UserRepository manages Firestore interactions for the User domain.
type UserRepository struct {
	client         *firestore.Client
	collectionName string
}

// NewUserRepository initializes a new UserRepository.
func NewUserRepository(client *firestore.Client, collectionName string) UserRepository {
	return UserRepository{
		client:         client,
		collectionName: collectionName,
	}
}

// convertUserToMap converts a domain.User to a map for Firestore storage.
func convertUserToMap(user domain.User) (map[string]interface{}, error) {
	if user.Username == nil || len(user.HashedPassword) < 1 {
		return nil, errors.New("missing required fields: username or password")
	}

	userMap := map[string]interface{}{
		"id":              user.Id,
		"hashed_password": user.HashedPassword,
	}

	if user.Username != nil {
		userMap["username"] = *user.Username
	}

	return userMap, nil
}

func (repo *UserRepository) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	userMap, err := convertUserToMap(user)
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to convert user: %w", err)
	}

	if _, err := repo.client.Collection(repo.collectionName).Doc(user.Id).Set(ctx, userMap); err != nil {
		return domain.User{}, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}
func (repo *UserRepository) GetUser(ctx context.Context, username string) (domain.User, error) {

	iter := repo.client.Collection(repo.collectionName).Where("username", "==", username).Limit(1).Documents(ctx)
	doc, err := iter.Next()
	if err == iterator.Done {
		return domain.User{}, errors.New("user not found")
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to get user by username: %w", err)
	}

	var user domain.User
	if err := doc.DataTo(&user); err != nil {
		return domain.User{}, fmt.Errorf("failed to parse user data: %w", err)
	}

	user.Id = doc.Ref.ID
	return user, nil
}
func (repo *UserRepository) UpdateUser(ctx context.Context, id string, user domain.User) (domain.User, error) {
	userMap, err := convertUserToMap(user)
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to convert user: %w", err)
	}

	if _, err := repo.client.Collection(repo.collectionName).Doc(id).Set(ctx, userMap, firestore.MergeAll); err != nil {
		return domain.User{}, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}
func (repo *UserRepository) DeleteUser(ctx context.Context, id string) error {
	if _, err := repo.client.Collection(repo.collectionName).Doc(id).Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
func (repo *UserRepository) GetUserPassword(ctx context.Context, username string) (domain.User, error) {
	iter := repo.client.Collection(repo.collectionName).Where("username", "==", username).Limit(1).Documents(ctx)
	doc, err := iter.Next()
	if err == iterator.Done {
		return domain.User{}, errors.New("user not found")
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to get user by username: %w", err)
	}

	var user domain.User
	if err := doc.DataTo(&user); err != nil {
		return domain.User{}, fmt.Errorf("failed to parse user data: %w", err)
	}

	user.Id = doc.Ref.ID
	return user, nil
}

// GetAllUsers retrieves users from Firestore with pagination.
func (repo *UserRepository) GetAllUsers(ctx context.Context, page, pageSize int) ([]domain.User, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	iter := repo.client.Collection(repo.collectionName).OrderBy("username", firestore.Asc).Offset(offset).Limit(pageSize).Documents(ctx)

	var users []domain.User
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate users: %w", err)
		}

		var user domain.User
		if err := doc.DataTo(&user); err != nil {
			return nil, fmt.Errorf("failed to parse user data: %w", err)
		}

		users = append(users, user)
	}

	return users, nil
}
