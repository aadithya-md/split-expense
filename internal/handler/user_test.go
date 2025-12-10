package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aadithya-md/split-expense/internal/repository"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(name, email string) (*repository.User, error) {
	args := m.Called(name, email)
	return args.Get(0).(*repository.User), args.Error(1)
}

func (m *MockUserService) GetUser(id int) (*repository.User, error) {
	args := m.Called(id)
	return args.Get(0).(*repository.User), args.Error(1)
}

func (m *MockUserService) GetUsersByEmails(emails []string) ([]*repository.User, error) {
	args := m.Called(emails)
	return args.Get(0).([]*repository.User), args.Error(1)
}

func TestUserHandler_CreateUserHandler(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService)

	// Test case 1: Successful user creation
	userToCreate := &repository.User{Name: "Test User", Email: "test@example.com"}
	expectedUser := &repository.User{ID: 1, Name: "Test User", Email: "test@example.com"}

	mockService.On("CreateUser", userToCreate.Name, userToCreate.Email).Return(expectedUser, nil).Once()

	body, _ := json.Marshal(userToCreate)
	req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateUserHandler(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var createdUser repository.User
	json.NewDecoder(rr.Body).Decode(&createdUser)
	assert.Equal(t, expectedUser, &createdUser)
	mockService.AssertExpectations(t)

	// Test case 2: Invalid request body
	req = httptest.NewRequest("POST", "/users", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	handler.CreateUserHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid request body")
	mockService.AssertNotCalled(t, "CreateUser")

	// Test case 3: Missing name or email
	body, _ = json.Marshal(struct{ Email string }{Email: "missingname@example.com"})
	req = httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	handler.CreateUserHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Name and Email are required")
	mockService.AssertNotCalled(t, "CreateUser")

	// Test case 4: Service error
	mockService.On("CreateUser", "Error User", "error@example.com").Return((*repository.User)(nil), fmt.Errorf("service error")).Once()

	body, _ = json.Marshal(struct{ Name, Email string }{Name: "Error User", Email: "error@example.com"})
	req = httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	handler.CreateUserHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "service error")
	mockService.AssertExpectations(t)
}

func TestUserHandler_GetUserHandler(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService)

	// Test case 1: Successful retrieval
	expectedUser := &repository.User{ID: 1, Name: "Test User", Email: "test@example.com"}
	mockService.On("GetUser", 1).Return(expectedUser, nil).Once()

	req := httptest.NewRequest("GET", "/users/1", nil)
	rr := httptest.NewRecorder()

	rtr := mux.NewRouter()
	rtr.HandleFunc("/users/{id}", handler.GetUserHandler).Methods("GET")
	rtr.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var retrievedUser repository.User
	json.NewDecoder(rr.Body).Decode(&retrievedUser)
	assert.Equal(t, expectedUser, &retrievedUser)
	mockService.AssertExpectations(t)

	// Test case 2: Invalid ID
	req = httptest.NewRequest("GET", "/users/abc", nil)
	rr = httptest.NewRecorder()

	rtr = mux.NewRouter()
	rtr.HandleFunc("/users/{id}", handler.GetUserHandler).Methods("GET")
	rtr.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid user ID")
	mockService.AssertNotCalled(t, "GetUser")

	// Test case 3: User not found
	mockService.On("GetUser", 99).Return((*repository.User)(nil), fmt.Errorf("user not found")).Once()

	req = httptest.NewRequest("GET", "/users/99", nil)
	rr = httptest.NewRecorder()

	rtr = mux.NewRouter()
	rtr.HandleFunc("/users/{id}", handler.GetUserHandler).Methods("GET")
	rtr.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "user not found")
	mockService.AssertExpectations(t)
}

func TestUserHandler_GetUserByEmailHandler(t *testing.T) {
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService)

	// Test case 1: Successful retrieval by email
	expectedUser := &repository.User{ID: 1, Name: "Test User", Email: "test@example.com"}
	mockService.On("GetUsersByEmails", []string{"test@example.com"}).Return([]*repository.User{expectedUser}, nil).Once()

	req := httptest.NewRequest("GET", "/users/by-email/test@example.com", nil)
	rr := httptest.NewRecorder()

	rtr := mux.NewRouter()
	rtr.HandleFunc("/users/by-email/{email}", handler.GetUserByEmailHandler).Methods("GET")
	rtr.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var retrievedUser repository.User
	json.NewDecoder(rr.Body).Decode(&retrievedUser)
	assert.Equal(t, expectedUser, &retrievedUser)
	mockService.AssertExpectations(t)

	// Test case 2: Missing email parameter
	{
		req := httptest.NewRequest("GET", "/users/by-email/", nil) // Path for mux to match, but email will be empty
		rr := httptest.NewRecorder()

		// Manually create a new router and set the vars, mimicking mux's behavior for an empty path param
		rtr := mux.NewRouter()
		rtr.HandleFunc("/users/by-email/{email}", handler.GetUserByEmailHandler).Methods("GET")

		// Create a mock request context with an empty "email" variable
		req = mux.SetURLVars(req, map[string]string{"email": ""})
		handler.GetUserByEmailHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Email parameter is required\n")
		mockService.AssertNotCalled(t, "GetUsersByEmails")
	}
	mockService.AssertNotCalled(t, "GetUsersByEmails")

	// Test case 3: User not found
	mockService.On("GetUsersByEmails", []string{"nonexistent@example.com"}).Return([]*repository.User{}, fmt.Errorf("user not found")).Once()

	req = httptest.NewRequest("GET", "/users/by-email/nonexistent@example.com", nil)
	rr = httptest.NewRecorder()

	rtr = mux.NewRouter()
	rtr.HandleFunc("/users/by-email/{email}", handler.GetUserByEmailHandler).Methods("GET")
	rtr.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "user not found")
	mockService.AssertExpectations(t)

	// Test case 4: Service error
	mockService.On("GetUsersByEmails", []string{"error@example.com"}).Return([]*repository.User{}, fmt.Errorf("service error")).Once()

	req = httptest.NewRequest("GET", "/users/by-email/error@example.com", nil)
	rr = httptest.NewRecorder()

	rtr = mux.NewRouter()
	rtr.HandleFunc("/users/by-email/{email}", handler.GetUserByEmailHandler).Methods("GET")
	rtr.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "service error")
	mockService.AssertExpectations(t)
}
