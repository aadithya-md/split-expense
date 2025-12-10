package service

import (
	"fmt"

	"github.com/aadithya-md/split-expense/internal/repository"
)

type UserService interface {
	CreateUser(name, email string) (*repository.User, error)
	GetUser(id int) (*repository.User, error)
	GetUserByEmail(email string) (*repository.User, error)
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

func (s *userService) GetUserByEmail(email string) (*repository.User, error) {
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email in service: %w", err)
	}
	return user, nil
}
