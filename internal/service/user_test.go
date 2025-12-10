package service

import (
	"fmt"
	"testing"

	"github.com/aadithya-md/split-expense/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateUser(user *repository.User) (*repository.User, error) {
	args := m.Called(user)
	return args.Get(0).(*repository.User), args.Error(1)
}

func (m *MockUserRepository) GetUser(id int) (*repository.User, error) {
	args := m.Called(id)
	return args.Get(0).(*repository.User), args.Error(1)
}

func (m *MockUserRepository) GetUsersByEmails(emails []string) ([]*repository.User, error) {
	args := m.Called(emails)
	return args.Get(0).([]*repository.User), args.Error(1)
}

func (m *MockUserRepository) GetUserByEmail(email string) (*repository.User, error) {
	args := m.Called(email)
	return args.Get(0).(*repository.User), args.Error(1)
}

func (m *MockUserRepository) GetUsersByIDs(ids []int) ([]*repository.User, error) {
	args := m.Called(ids)
	return args.Get(0).([]*repository.User), args.Error(1)
}

func TestUserService_CreateUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	// Test case 1: Successful user creation
	expectedUser := &repository.User{ID: 1, Name: "Test User", Email: "test@example.com"}
	mockRepo.On("CreateUser", &repository.User{Name: "Test User", Email: "test@example.com"}).Return(expectedUser, nil).Once()

	createdUser, err := userService.CreateUser("Test User", "test@example.com")
	assert.Nil(t, err)
	assert.Equal(t, expectedUser, createdUser)
	mockRepo.AssertExpectations(t)

	// Test case 2: Error from repository
	mockRepo.On("CreateUser", &repository.User{Name: "Error User", Email: "error@example.com"}).Return((*repository.User)(nil), fmt.Errorf("repo error")).Once()

	createdUser, err = userService.CreateUser("Error User", "error@example.com")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "repo error")
	assert.Nil(t, createdUser)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	// Test case 1: Successful retrieval
	expectedUser := &repository.User{ID: 1, Name: "Test User", Email: "test@example.com"}
	mockRepo.On("GetUser", 1).Return(expectedUser, nil).Once()

	user, err := userService.GetUser(1)
	assert.Nil(t, err)
	assert.Equal(t, expectedUser, user)
	mockRepo.AssertExpectations(t)

	// Test case 2: User not found
	mockRepo.On("GetUser", 99).Return((*repository.User)(nil), fmt.Errorf("user not found")).Once()

	user, err = userService.GetUser(99)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "user not found")
	assert.Nil(t, user)
	mockRepo.AssertExpectations(t)

	// Test case 3: Error from repository
	mockRepo.On("GetUser", 2).Return((*repository.User)(nil), fmt.Errorf("repo error")).Once()

	user, err = userService.GetUser(2)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "repo error")
	assert.Nil(t, user)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetUserByEmail(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	// Test case 1: Successful retrieval by email
	expectedUser := &repository.User{ID: 1, Name: "Test User", Email: "test@example.com"}
	mockRepo.On("GetUsersByEmails", []string{"test@example.com"}).Return([]*repository.User{expectedUser}, nil).Once()

	users, err := userService.GetUsersByEmails([]string{"test@example.com"})
	assert.Nil(t, err)
	assert.NotNil(t, users)
	assert.Equal(t, 1, len(users))
	assert.Equal(t, expectedUser, users[0])
	mockRepo.AssertExpectations(t)

	// Test case 2: User not found by email
	mockRepo.On("GetUsersByEmails", []string{"nonexistent@example.com"}).Return([]*repository.User{}, fmt.Errorf("some users not found for emails: nonexistent@example.com")).Once()

	users, err = userService.GetUsersByEmails([]string{"nonexistent@example.com"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "some users not found")
	assert.Empty(t, users)
	mockRepo.AssertExpectations(t)

	// Test case 3: Error from repository
	mockRepo.On("GetUsersByEmails", []string{"error@example.com"}).Return([]*repository.User{}, fmt.Errorf("repo error")).Once()

	users, err = userService.GetUsersByEmails([]string{"error@example.com"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "repo error")
	assert.Empty(t, users)
	mockRepo.AssertExpectations(t)
}
