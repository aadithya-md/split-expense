package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aadithya-md/split-expense/internal/repository"
	"github.com/aadithya-md/split-expense/internal/service"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockExpenseService struct {
	mock.Mock
}

func (m *MockExpenseService) CreateExpense(req service.CreateExpenseRequest) (*repository.Expense, error) {
	args := m.Called(req)
	return args.Get(0).(*repository.Expense), args.Error(1)
}

func (m *MockExpenseService) GetExpense(id int) (*repository.Expense, error) {
	args := m.Called(id)
	return args.Get(0).(*repository.Expense), args.Error(1)
}

func (m *MockExpenseService) GetExpensesForUser(userEmail string) ([]repository.UserExpenseView, error) {
	args := m.Called(userEmail)
	return args.Get(0).([]repository.UserExpenseView), args.Error(1)
}

func (m *MockExpenseService) GetOutstandingBalancesForUser(userEmail string) ([]service.UserBalanceView, error) {
	args := m.Called(userEmail)
	return args.Get(0).([]service.UserBalanceView), args.Error(1)
}

func TestExpenseHandler_CreateExpenseHandler(t *testing.T) {
	mockService := new(MockExpenseService)
	expenseHandler := NewExpenseHandler(mockService)

	// Test case 1: Successful Equal Split expense creation
	{ // Block for scoping
		requestBody := service.CreateExpenseRequest{
			Description:    "Team Lunch (Equal)",
			TotalAmount:    150.00,
			CreatedByEmail: "alice@example.com",
			SplitMethod:    service.SplitMethodEqual,
			EqualSplits: []service.EqualSplitRequest{
				{UserEmail: "alice@example.com", AmountPaid: 150.00},
				{UserEmail: "bob@example.com", AmountPaid: 0.00},
				{UserEmail: "charlie@example.com", AmountPaid: 0.00},
			},
		}
		expectedExpense := &repository.Expense{
			ID:          1,
			Description: requestBody.Description,
			TotalAmount: requestBody.TotalAmount,
			CreatedBy:   1,
		}

		mockService.On("CreateExpense", requestBody).Return(expectedExpense, nil).Once()

		reqBodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/expenses", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/expenses", expenseHandler.CreateExpenseHandler).Methods("POST")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		expectedResponseBytes, _ := json.Marshal(expectedExpense)
		assert.JSONEq(t, string(expectedResponseBytes), rr.Body.String())
		mockService.AssertExpectations(t)
	}

	// Test case 2: Invalid request body (missing fields - description)
	{ // Block for scoping
		reqBodyBytes := []byte(`{"total_amount":100,"created_by_email":"alice@example.com","split_method":"equal","equal_splits":[]}`)
		req := httptest.NewRequest("POST", "/expenses", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/expenses", expenseHandler.CreateExpenseHandler).Methods("POST")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "description, total_amount, created_by, and split_method are required")
		mockService.AssertNotCalled(t, "CreateExpense")
	}

	// Test case 3: Service returns an error
	{ // Block for scoping
		requestBody := service.CreateExpenseRequest{
			Description:    "Service Error Test",
			TotalAmount:    100.00,
			CreatedByEmail: "alice@example.com",
			SplitMethod:    service.SplitMethodEqual,
			EqualSplits: []service.EqualSplitRequest{
				{UserEmail: "alice@example.com", AmountPaid: 100.00},
			},
		}
		mockService.On("CreateExpense", requestBody).Return((*repository.Expense)(nil), errors.New("failed to create expense in service")).Once()

		reqBodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/expenses", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/expenses", expenseHandler.CreateExpenseHandler).Methods("POST")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "failed to create expense in service")
		mockService.AssertExpectations(t)
	}

	// Test case 4: Percentage Split with invalid total percentage (validation error)
	{ // Block for scoping
		requestBody := service.CreateExpenseRequest{
			Description:    "Invalid Percentage Test",
			TotalAmount:    100.00,
			CreatedByEmail: "alice@example.com",
			SplitMethod:    service.SplitMethodPercentage,
			PercentageSplits: []service.PercentageSplitRequest{
				{UserEmail: "alice@example.com", Percentage: 60, AmountPaid: 100.00},
				{UserEmail: "bob@example.com", Percentage: 30, AmountPaid: 0.00},
			},
		}

		reqBodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/expenses", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/expenses", expenseHandler.CreateExpenseHandler).Methods("POST")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "total percentage across all splits must be 100%")
		mockService.AssertNotCalled(t, "CreateExpense")
	}

	// Test case 5: Manual Split with amount_owed mismatch (validation error)
	{ // Block for scoping
		requestBody := service.CreateExpenseRequest{
			Description:    "Invalid Manual Test",
			TotalAmount:    100.00,
			CreatedByEmail: "alice@example.com",
			SplitMethod:    service.SplitMethodManual,
			ManualSplits: []service.ManualSplitRequest{
				{UserEmail: "alice@example.com", AmountOwed: 60.00, AmountPaid: 100.00},
				{UserEmail: "bob@example.com", AmountOwed: 30.00, AmountPaid: 0.00},
			},
		}

		reqBodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/expenses", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/expenses", expenseHandler.CreateExpenseHandler).Methods("POST")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "total amount owed across all splits (90.00) does not match total expense amount (100.00)")
		mockService.AssertNotCalled(t, "CreateExpense")
	}

	// Test case 6: Duplicate email in Equal Splits (validation error)
	{ // Block for scoping
		requestBody := service.CreateExpenseRequest{
			Description:    "Duplicate Email Test",
			TotalAmount:    100.00,
			CreatedByEmail: "alice@example.com",
			SplitMethod:    service.SplitMethodEqual,
			EqualSplits: []service.EqualSplitRequest{
				{UserEmail: "alice@example.com", AmountPaid: 50.00},
				{UserEmail: "alice@example.com", AmountPaid: 50.00},
			},
		}

		reqBodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/expenses", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/expenses", expenseHandler.CreateExpenseHandler).Methods("POST")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "duplicate email found in splits: alice@example.com")
		mockService.AssertNotCalled(t, "CreateExpense")
	}

	// Test case 7: Creator not in splits (validation error)
	{ // Block for scoping
		requestBody := service.CreateExpenseRequest{
			Description:    "Creator Not in Splits Test",
			TotalAmount:    100.00,
			CreatedByEmail: "alice@example.com",
			SplitMethod:    service.SplitMethodEqual,
			EqualSplits: []service.EqualSplitRequest{
				{UserEmail: "bob@example.com", AmountPaid: 100.00},
			},
		}

		reqBodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/expenses", bytes.NewBuffer(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/expenses", expenseHandler.CreateExpenseHandler).Methods("POST")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "created_by user (alice@example.com) must be included in the split participants")
		mockService.AssertNotCalled(t, "CreateExpense")
	}
}

func TestExpenseHandler_GetExpensesForUserHandler(t *testing.T) {
	mockService := new(MockExpenseService)
	expenseHandler := NewExpenseHandler(mockService)

	// Test Case 1: Successful retrieval of expenses for a user
	{
		userEmail := "alice@example.com"
		expectedExpenses := []repository.UserExpenseView{
			{Date: time.Now(), Tag: "Food", Description: "Dinner", TotalAmount: 50.00, Share: 25.00},
			{Date: time.Now().Add(-24 * time.Hour), Tag: "Transport", Description: "Uber", TotalAmount: 15.00, Share: 7.50},
		}

		mockService.On("GetExpensesForUser", userEmail).Return(expectedExpenses, nil).Once()

		req := httptest.NewRequest("GET", "/expenses/by-user/"+userEmail, nil)
		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/expenses/by-user/{email}", expenseHandler.GetExpensesForUserHandler).Methods("GET")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var actualExpenses []repository.UserExpenseView
		json.NewDecoder(rr.Body).Decode(&actualExpenses)
		// Compare fields individually due to time.Time comparison issues
		assert.Equal(t, len(expectedExpenses), len(actualExpenses))
		if len(expectedExpenses) == len(actualExpenses) {
			for i := range expectedExpenses {
				assert.WithinDuration(t, expectedExpenses[i].Date, actualExpenses[i].Date, time.Second)
				assert.Equal(t, expectedExpenses[i].Tag, actualExpenses[i].Tag)
				assert.Equal(t, expectedExpenses[i].Description, actualExpenses[i].Description)
				assert.Equal(t, expectedExpenses[i].TotalAmount, actualExpenses[i].TotalAmount)
				assert.Equal(t, expectedExpenses[i].Share, actualExpenses[i].Share)
			}
		}
		mockService.AssertExpectations(t)
	}

	// Test Case 2: User not found / Service returns error
	{
		userEmail := "nonexistent@example.com"
		mockService.On("GetExpensesForUser", userEmail).Return([]repository.UserExpenseView{}, errors.New("user not found")).Once()

		req := httptest.NewRequest("GET", "/expenses/by-user/"+userEmail, nil)
		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/expenses/by-user/{email}", expenseHandler.GetExpensesForUserHandler).Methods("GET")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		//		assert.Contains(t, rr.Body.String(), "Failed to retrieve expenses")
		mockService.AssertExpectations(t)
	}
}

func TestExpenseHandler_GetOutstandingBalancesHandler(t *testing.T) {
	mockService := new(MockExpenseService)
	expenseHandler := NewExpenseHandler(mockService)

	// Test Case 1: Successful retrieval of outstanding balances for a user
	{
		userEmail := "alice@example.com"
		fixedTime := time.Date(2023, 5, 10, 12, 0, 0, 0, time.UTC)
		expectedBalances := []service.UserBalanceView{
			{WithUserEmail: "bob@example.com", WithUserName: "Bob", Amount: 15.00, LastUpdated: fixedTime},
			{WithUserEmail: "charlie@example.com", WithUserName: "Charlie", Amount: -10.00, LastUpdated: fixedTime},
		}

		mockService.On("GetOutstandingBalancesForUser", userEmail).Return(expectedBalances, nil).Once()

		req := httptest.NewRequest("GET", "/balances/by-user/"+userEmail, nil)
		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/balances/by-user/{email}", expenseHandler.GetOutstandingBalancesHandler).Methods("GET")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var actualBalances []service.UserBalanceView
		json.NewDecoder(rr.Body).Decode(&actualBalances)
		assert.Equal(t, len(expectedBalances), len(actualBalances))
		if len(expectedBalances) == len(actualBalances) {
			for i := range expectedBalances {
				assert.Equal(t, expectedBalances[i].WithUserEmail, actualBalances[i].WithUserEmail)
				assert.Equal(t, expectedBalances[i].WithUserName, actualBalances[i].WithUserName)
				assert.Equal(t, expectedBalances[i].Amount, actualBalances[i].Amount)
				assert.WithinDuration(t, expectedBalances[i].LastUpdated, actualBalances[i].LastUpdated, time.Second)
			}
		}
		mockService.AssertExpectations(t)
	}

	// Test Case 2: User not found / Service returns error
	{
		userEmail := "nonexistent@example.com"
		mockService.On("GetOutstandingBalancesForUser", userEmail).Return([]service.UserBalanceView{}, errors.New("user not found")).Once()

		req := httptest.NewRequest("GET", "/balances/by-user/"+userEmail, nil)
		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/balances/by-user/{email}", expenseHandler.GetOutstandingBalancesHandler).Methods("GET")
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		//		assert.Contains(t, rr.Body.String(), "Failed to retrieve outstanding balances")
		mockService.AssertExpectations(t)
	}
}
