package service

import (
	"database/sql"
	"testing"
	"time"

	"github.com/aadithya-md/split-expense/internal/repository"
	"github.com/aadithya-md/split-expense/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockExpenseRepository struct {
	mock.Mock
}

func (m *MockExpenseRepository) CreateExpense(expense *repository.Expense, splits []repository.ExpenseSplit, balanceUpdates []repository.BalanceUpdate) (*repository.Expense, error) {
	args := m.Called(expense, splits, balanceUpdates)
	return args.Get(0).(*repository.Expense), args.Error(1)
}

func (m *MockExpenseRepository) GetExpense(id int) (*repository.Expense, error) {
	args := m.Called(id)
	return args.Get(0).(*repository.Expense), args.Error(1)
}

func (m *MockExpenseRepository) GetExpensesByUserID(userID int) ([]repository.UserExpenseView, error) {
	args := m.Called(userID)
	return args.Get(0).([]repository.UserExpenseView), args.Error(1)
}

// This mock should be defined in a separate file if used by multiple tests.
// For now, it's here for simplicity.
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

func (m *MockUserService) GetUsersByIDs(ids []int) ([]*repository.User, error) {
	args := m.Called(ids)
	return args.Get(0).([]*repository.User), args.Error(1)
}

type MockBalanceRepository struct {
	mock.Mock
}

func (m *MockBalanceRepository) UpdateBalance(tx *sql.Tx, user1ID, user2ID int, amount float64) error {
	args := m.Called(tx, user1ID, user2ID, amount)
	return args.Error(0)
}

func (m *MockBalanceRepository) GetBalancesByUserID(userID int) ([]repository.Balance, error) {
	args := m.Called(userID)
	return args.Get(0).([]repository.Balance), args.Error(1)
}

func TestExpenseService_CreateExpense(t *testing.T) {
	expenseRepo := new(MockExpenseRepository)
	userService := new(MockUserService)
	balanceRepo := new(MockBalanceRepository)
	expenseService := NewExpenseService(expenseRepo, userService, balanceRepo)

	// Setup common users for all tests
	alice := &repository.User{ID: 1, Name: "Alice", Email: "alice@example.com"}
	bob := &repository.User{ID: 2, Name: "Bob", Email: "bob@example.com"}
	charlie := &repository.User{ID: 3, Name: "Charlie", Email: "charlie@example.com"}
	usersMap := map[string]*repository.User{
		alice.Email:   alice,
		bob.Email:     bob,
		charlie.Email: charlie,
	}

	// Helper to create expected splits for comparison (ignoring AmountPaid and CreatedBy for simplicity here)
	createExpectedSplits := func(totalAmount float64, splitMethod SplitMethodType, participants map[string]*repository.User, req CreateExpenseRequest) []repository.ExpenseSplit {
		splits := make([]repository.ExpenseSplit, 0)
		switch splitMethod {
		case SplitMethodEqual:
			amountPerUser := util.RoundToTwoDecimalPlaces(totalAmount / float64(len(req.EqualSplits)))
			for i, es := range req.EqualSplits {
				owed := amountPerUser
				if i == 0 {
					owed = util.RoundToTwoDecimalPlaces(totalAmount - (amountPerUser * float64(len(req.EqualSplits)-1)))
				}
				splits = append(splits, repository.ExpenseSplit{UserID: participants[es.UserEmail].ID, AmountOwed: owed, AmountPaid: util.RoundToTwoDecimalPlaces(es.AmountPaid)})
			}
		case SplitMethodPercentage:
			var currentTotalOwed float64
			for _, ps := range req.PercentageSplits {
				owed := util.RoundToTwoDecimalPlaces(totalAmount * (ps.Percentage / 100))
				splits = append(splits, repository.ExpenseSplit{UserID: participants[ps.UserEmail].ID, AmountOwed: owed, AmountPaid: util.RoundToTwoDecimalPlaces(ps.AmountPaid)})
				currentTotalOwed += owed
			}
			diff := util.RoundToTwoDecimalPlaces(totalAmount - currentTotalOwed)
			if diff != 0 && len(splits) > 0 {
				splits[0].AmountOwed = util.RoundToTwoDecimalPlaces(splits[0].AmountOwed + diff)
			}
		case SplitMethodManual:
			for _, ms := range req.ManualSplits {
				splits = append(splits, repository.ExpenseSplit{UserID: participants[ms.UserEmail].ID, AmountOwed: util.RoundToTwoDecimalPlaces(ms.AmountOwed), AmountPaid: util.RoundToTwoDecimalPlaces(ms.AmountPaid)})
			}
		}
		return splits
	}

	// Test case 1: Successful Equal Split
	{ // Use a block to avoid variable shadowing
		req := CreateExpenseRequest{
			Description:    "Equal Split Test",
			TotalAmount:    30.00,
			CreatedByEmail: "alice@example.com",
			SplitMethod:    SplitMethodEqual,
			EqualSplits: []EqualSplitRequest{
				{UserEmail: "alice@example.com", AmountPaid: 10.00},
				{UserEmail: "bob@example.com", AmountPaid: 10.00},
				{UserEmail: "charlie@example.com", AmountPaid: 10.00},
			},
		}
		userService.On("GetUsersByEmails", mock.AnythingOfType("[]string")).Return([]*repository.User{alice, bob, charlie}, nil).Once()

		expectedExpense := &repository.Expense{ID: 1, Description: req.Description, TotalAmount: req.TotalAmount, CreatedBy: alice.ID, CreatedAt: time.Now()}
		expectedSplits := createExpectedSplits(req.TotalAmount, req.SplitMethod, usersMap, req)
		expenseRepo.On("CreateExpense", mock.AnythingOfType("*repository.Expense"), expectedSplits, mock.Anything).Return(expectedExpense, nil).Once()

		createdExpense, err := expenseService.CreateExpense(req)
		assert.Nil(t, err)
		assert.Equal(t, expectedExpense.Description, createdExpense.Description)
		assert.Equal(t, expectedExpense.TotalAmount, createdExpense.TotalAmount)
		assert.Equal(t, expectedExpense.CreatedBy, createdExpense.CreatedBy)
		assert.NotZero(t, createdExpense.CreatedAt) // CreatedAt is set by repo now
		expenseRepo.AssertExpectations(t)
		userService.AssertExpectations(t)
	}

	// Test case 2: User not found during email mapping
	{ // Use a block to avoid variable shadowing
		req := CreateExpenseRequest{
			Description:    "User Not Found Test",
			TotalAmount:    30.00,
			CreatedByEmail: "nonexistent@example.com",
			SplitMethod:    SplitMethodEqual,
			EqualSplits: []EqualSplitRequest{
				{UserEmail: "nonexistent@example.com", AmountPaid: 30.00},
			},
		}
		userService.On("GetUsersByEmails", mock.AnythingOfType("[]string")).Return([]*repository.User{}, nil).Once() // Return empty slice, no error

		createdExpense, err := expenseService.CreateExpense(req)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "created_by user not found")
		assert.Nil(t, createdExpense)
		expenseRepo.AssertNotCalled(t, "CreateExpense")
		userService.AssertExpectations(t)
	}

	// Test case 3: Total amount paid mismatch
	{ // Use a block to avoid variable shadowing
		req := CreateExpenseRequest{
			Description:    "Paid Mismatch Test",
			TotalAmount:    30.00,
			CreatedByEmail: "alice@example.com",
			SplitMethod:    SplitMethodEqual,
			EqualSplits: []EqualSplitRequest{
				{UserEmail: "alice@example.com", AmountPaid: 15.00},
				{UserEmail: "bob@example.com", AmountPaid: 10.00},
			},
		}
		userService.On("GetUsersByEmails", mock.AnythingOfType("[]string")).Return([]*repository.User{alice, bob}, nil).Once()

		createdExpense, err := expenseService.CreateExpense(req)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "total amount paid across all splits (25.00) does not match total expense amount (30.00)")
		assert.Nil(t, createdExpense)
		expenseRepo.AssertNotCalled(t, "CreateExpense")
		userService.AssertExpectations(t)
	}

	// Test case 4: Percentage Split Success
	{ // Use a block to avoid variable shadowing
		req := CreateExpenseRequest{
			Description:    "Percentage Split Test",
			TotalAmount:    100.00,
			CreatedByEmail: "alice@example.com",
			SplitMethod:    SplitMethodPercentage,
			PercentageSplits: []PercentageSplitRequest{
				{UserEmail: "alice@example.com", Percentage: 50, AmountPaid: 70.00},
				{UserEmail: "bob@example.com", Percentage: 30, AmountPaid: 30.00},
				{UserEmail: "charlie@example.com", Percentage: 20, AmountPaid: 0.00},
			},
		}
		userService.On("GetUsersByEmails", mock.AnythingOfType("[]string")).Return([]*repository.User{alice, bob, charlie}, nil).Once()

		expectedExpense := &repository.Expense{ID: 2, Description: req.Description, TotalAmount: req.TotalAmount, CreatedBy: alice.ID, CreatedAt: time.Now()}
		expectedSplits := createExpectedSplits(req.TotalAmount, req.SplitMethod, usersMap, req)
		expenseRepo.On("CreateExpense", mock.AnythingOfType("*repository.Expense"), expectedSplits, mock.Anything).Return(expectedExpense, nil).Once()

		createdExpense, err := expenseService.CreateExpense(req)
		assert.Nil(t, err)
		assert.Equal(t, expectedExpense.Description, createdExpense.Description)
		assert.Equal(t, expectedExpense.TotalAmount, createdExpense.TotalAmount)
		assert.Equal(t, expectedExpense.CreatedBy, createdExpense.CreatedBy)
		assert.NotZero(t, createdExpense.CreatedAt)
		expenseRepo.AssertExpectations(t)
		userService.AssertExpectations(t)
	}

	// Test case 5: Manual Split Success
	{ // Use a block to avoid variable shadowing
		req := CreateExpenseRequest{
			Description:    "Manual Split Test",
			TotalAmount:    50.00,
			CreatedByEmail: "bob@example.com",
			SplitMethod:    SplitMethodManual,
			ManualSplits: []ManualSplitRequest{
				{UserEmail: "alice@example.com", AmountOwed: 10.00, AmountPaid: 0.00},
				{UserEmail: "bob@example.com", AmountOwed: 20.00, AmountPaid: 50.00},
				{UserEmail: "charlie@example.com", AmountOwed: 20.00, AmountPaid: 0.00},
			},
		}
		userService.On("GetUsersByEmails", mock.AnythingOfType("[]string")).Return([]*repository.User{alice, bob, charlie}, nil).Once()

		expectedExpense := &repository.Expense{ID: 3, Description: req.Description, TotalAmount: req.TotalAmount, CreatedBy: bob.ID, CreatedAt: time.Now()}
		expectedSplits := createExpectedSplits(req.TotalAmount, req.SplitMethod, usersMap, req)
		expenseRepo.On("CreateExpense", mock.AnythingOfType("*repository.Expense"), expectedSplits, mock.Anything).Return(expectedExpense, nil).Once()

		createdExpense, err := expenseService.CreateExpense(req)
		assert.Nil(t, err)
		assert.Equal(t, expectedExpense.Description, createdExpense.Description)
		assert.Equal(t, expectedExpense.TotalAmount, createdExpense.TotalAmount)
		assert.Equal(t, expectedExpense.CreatedBy, createdExpense.CreatedBy)
		assert.NotZero(t, createdExpense.CreatedAt)
		expenseRepo.AssertExpectations(t)
		userService.AssertExpectations(t)
	}

	// Test case 6: Invalid percentage split total
	{ // Use a block to avoid variable shadowing
		req := CreateExpenseRequest{
			Description:    "Invalid Percentage Test",
			TotalAmount:    100.00,
			CreatedByEmail: "alice@example.com",
			SplitMethod:    SplitMethodPercentage,
			PercentageSplits: []PercentageSplitRequest{
				{UserEmail: "alice@example.com", Percentage: 60, AmountPaid: 100.00},
				{UserEmail: "bob@example.com", Percentage: 30, AmountPaid: 0.00},
			},
		}
		userService.On("GetUsersByEmails", mock.AnythingOfType("[]string")).Return([]*repository.User{alice, bob}, nil).Once()

		createdExpense, err := expenseService.CreateExpense(req)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "percentage split total must be 100%")
		assert.Nil(t, createdExpense)
		expenseRepo.AssertNotCalled(t, "CreateExpense")
		userService.AssertExpectations(t)
	}

	// Test case 7: Invalid manual split total
	{ // Use a block to avoid variable shadowing
		req := CreateExpenseRequest{
			Description:    "Invalid Manual Test",
			TotalAmount:    100.00,
			CreatedByEmail: "alice@example.com",
			SplitMethod:    SplitMethodManual,
			ManualSplits: []ManualSplitRequest{
				{UserEmail: "alice@example.com", AmountOwed: 60.00, AmountPaid: 100.00},
				{UserEmail: "bob@example.com", AmountOwed: 30.00, AmountPaid: 0.00},
			},
		}
		userService.On("GetUsersByEmails", mock.AnythingOfType("[]string")).Return([]*repository.User{alice, bob}, nil).Once()

		createdExpense, err := expenseService.CreateExpense(req)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "manual split amounts (90.00) must sum up to total amount (100.00)")
		assert.Nil(t, createdExpense)
		expenseRepo.AssertNotCalled(t, "CreateExpense")
		userService.AssertExpectations(t)
	}
}

func TestExpenseService_GetExpensesForUser(t *testing.T) {
	expenseRepo := new(MockExpenseRepository)
	userService := new(MockUserService)
	balanceRepo := new(MockBalanceRepository)
	expenseService := NewExpenseService(expenseRepo, userService, balanceRepo)

	alice := &repository.User{ID: 1, Name: "Alice", Email: "alice@example.com"}

	// Test case for GetExpensesForUser
	{
		userEmail := "alice@example.com"
		expectedUserExpenses := []repository.UserExpenseView{
			{Date: time.Now(), Tag: "Food", Description: "Dinner", TotalAmount: 50.00, Share: -20.00},
		}

		userService.On("GetUsersByEmails", []string{userEmail}).Return([]*repository.User{alice}, nil).Once()
		expenseRepo.On("GetExpensesByUserID", alice.ID).Return(expectedUserExpenses, nil).Once()

		expenses, err := expenseService.GetExpensesForUser(userEmail)
		assert.Nil(t, err)
		assert.NotNil(t, expenses)
		assert.Equal(t, expectedUserExpenses, expenses)
		userService.AssertExpectations(t)
		expenseRepo.AssertExpectations(t)
	}
}

func TestExpenseService_GetOutstandingBalancesForUser(t *testing.T) {
	expenseRepo := new(MockExpenseRepository)
	userService := new(MockUserService)
	balanceRepo := new(MockBalanceRepository)
	expenseService := NewExpenseService(expenseRepo, userService, balanceRepo)

	alice := &repository.User{ID: 1, Name: "Alice", Email: "alice@example.com"}
	bob := &repository.User{ID: 2, Name: "Bob", Email: "bob@example.com"}
	charlie := &repository.User{ID: 3, Name: "Charlie", Email: "charlie@example.com"}

	// Test case for GetOutstandingBalancesForUser
	{
		userEmail := "alice@example.com"
		now := time.Now()
		expectedBalances := []repository.Balance{
			{User1ID: alice.ID, User2ID: bob.ID, Balance: 15.00, LastUpdated: now},
			{User1ID: alice.ID, User2ID: charlie.ID, Balance: -10.00, LastUpdated: now},
		}
		expectedUserBalances := []UserBalanceView{
			{WithUserEmail: "bob@example.com", WithUserName: "Bob", Amount: 15.00, LastUpdated: now},
			{WithUserEmail: "charlie@example.com", WithUserName: "Charlie", Amount: -10.00, LastUpdated: now},
		}

		userService.On("GetUsersByEmails", []string{userEmail}).Return([]*repository.User{alice}, nil).Once()
		balanceRepo.On("GetBalancesByUserID", alice.ID).Return(expectedBalances, nil).Once()
		userService.On("GetUsersByIDs", []int{bob.ID, charlie.ID}).Return([]*repository.User{bob, charlie}, nil).Once()

		balances, err := expenseService.GetOutstandingBalancesForUser(userEmail)
		assert.Nil(t, err)
		assert.NotNil(t, balances)
		assert.Equal(t, expectedUserBalances, balances)
		userService.AssertExpectations(t)
		balanceRepo.AssertExpectations(t)
	}
}
