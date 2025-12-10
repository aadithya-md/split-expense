package service

import (
	"fmt"

	"github.com/aadithya-md/split-expense/internal/repository"
)

type UserService interface {
	CreateUser(name, email string) (*repository.User, error)
	GetUser(id int) (*repository.User, error)
	GetUsersByEmails(emails []string) ([]*repository.User, error)
	GetUsersByIDs(ids []int) ([]*repository.User, error)
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) CreateUser(name, email string) (*repository.User, error) {
	user := &repository.User{
		Name:  name,
		Email: email,
	}

	createdUser, err := s.repo.CreateUser(user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user in service: %w", err)
	}

	return createdUser, nil
}

func (s *userService) GetUser(id int) (*repository.User, error) {
	user, err := s.repo.GetUser(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user in service: %w", err)
	}
	return user, nil
}

func (s *userService) GetUsersByEmails(emails []string) ([]*repository.User, error) {
	users, err := s.repo.GetUsersByEmails(emails)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by emails in service: %w", err)
	}
	return users, nil
}

func (s *userService) GetUsersByIDs(ids []int) ([]*repository.User, error) {
	users, err := s.repo.GetUsersByIDs(ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by IDs in service: %w", err)
	}
	return users, nil
}
