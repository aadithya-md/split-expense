package service

import (
	"fmt"

	"github.com/aadithya-md/split-expense/internal/repository"
	"github.com/aadithya-md/split-expense/internal/util"
)

// SplitMethodType defines the allowed types of expense splitting.
type SplitMethodType string

const (
	SplitMethodEqual      SplitMethodType = "equal"
	SplitMethodPercentage SplitMethodType = "percentage"
	SplitMethodManual     SplitMethodType = "manual"
)

type EqualSplitRequest struct {
	UserEmail  string  `json:"user_email"`
	UserID     int     `json:"-"` // Populated by service layer
	AmountPaid float64 `json:"amount_paid,omitempty"`
}

type PercentageSplitRequest struct {
	UserEmail  string  `json:"user_email"`
	UserID     int     `json:"-"` // Populated by service layer
	Percentage float64 `json:"percentage"`
	AmountPaid float64 `json:"amount_paid,omitempty"`
}

type ManualSplitRequest struct {
	UserEmail  string  `json:"user_email"`
	UserID     int     `json:"-"` // Populated by service layer
	AmountOwed float64 `json:"amount_owed"`
	AmountPaid float64 `json:"amount_paid,omitempty"`
}

type CreateExpenseRequest struct {
	Description      string                   `json:"description"`
	Tag              string                   `json:"tag"`
	TotalAmount      float64                  `json:"total_amount"`
	CreatedByEmail   string                   `json:"created_by_email"`
	CreatedByID      int                      `json:"-"`            // Populated by service layer
	SplitMethod      SplitMethodType          `json:"split_method"` // "equal", "percentage", "manual"
	EqualSplits      []EqualSplitRequest      `json:"equal_splits,omitempty"`
	PercentageSplits []PercentageSplitRequest `json:"percentage_splits,omitempty"`
	ManualSplits     []ManualSplitRequest     `json:"manual_splits,omitempty"`
}

type ExpenseService interface {
	CreateExpense(req CreateExpenseRequest) (*repository.Expense, error)
	GetExpensesForUser(userEmail string) ([]repository.UserExpenseView, error)
}

type expenseService struct {
	expenseRepo repository.ExpenseRepository
	userService UserService
}

func NewExpenseService(expenseRepo repository.ExpenseRepository, userService UserService) ExpenseService {
	return &expenseService{expenseRepo: expenseRepo, userService: userService}
}

func (s *expenseService) calculateExpenseSplits(req CreateExpenseRequest) ([]repository.ExpenseSplit, error) {
	strategy, err := getSplitStrategy(req.SplitMethod)
	if err != nil {
		return nil, err
	}

	splits, err := strategy.CalculateSplits(req) // No longer passing usersMap
	if err != nil {
		return nil, err
	}

	return splits, nil
}

// resolveUserEmailsToIDs gathers all unique emails from the request, fetches users in a batch,
// and populates the corresponding UserID fields within the CreateExpenseRequest.
func (s *expenseService) resolveUserEmailsToIDs(req *CreateExpenseRequest) error {
	// Gather all unique emails from the request using Set
	emailsToFetch := util.NewSet[string]()
	emailsToFetch.Add(req.CreatedByEmail) // Add creator's email

	switch req.SplitMethod {
	case SplitMethodEqual:
		for _, es := range req.EqualSplits {
			emailsToFetch.Add(es.UserEmail)
		}
	case SplitMethodPercentage:
		for _, ps := range req.PercentageSplits {
			emailsToFetch.Add(ps.UserEmail)
		}
	case SplitMethodManual:
		for _, ms := range req.ManualSplits {
			emailsToFetch.Add(ms.UserEmail)
		}
	}

	emailList := emailsToFetch.ToList()

	// Fetch all users in a single batch call
	usersSlice, err := s.userService.GetUsersByEmails(emailList)
	if err != nil {
		return fmt.Errorf("failed to fetch users for expense: %w", err)
	}

	// Convert slice to map for efficient lookup
	resolvedUsersMap := make(map[string]*repository.User, len(usersSlice))
	for _, user := range usersSlice {
		resolvedUsersMap[user.Email] = user
	}

	// Populate CreatedByID
	creator, ok := resolvedUsersMap[req.CreatedByEmail]
	if !ok {
		return fmt.Errorf("created_by user not found: %s", req.CreatedByEmail)
	}
	req.CreatedByID = creator.ID

	// Populate UserID for all splits
	switch req.SplitMethod {
	case SplitMethodEqual:
		for i, es := range req.EqualSplits {
			user, ok := resolvedUsersMap[es.UserEmail]
			if !ok {
				return fmt.Errorf("equal split participant not found: %s", es.UserEmail)
			}
			req.EqualSplits[i].UserID = user.ID
		}
	case SplitMethodPercentage:
		for i, ps := range req.PercentageSplits {
			user, ok := resolvedUsersMap[ps.UserEmail]
			if !ok {
				return fmt.Errorf("percentage split participant not found: %s", ps.UserEmail)
			}
			req.PercentageSplits[i].UserID = user.ID
		}
	case SplitMethodManual:
		for i, ms := range req.ManualSplits {
			user, ok := resolvedUsersMap[ms.UserEmail]
			if !ok {
				return fmt.Errorf("manual split participant not found: %s", ms.UserEmail)
			}
			req.ManualSplits[i].UserID = user.ID
		}
	}

	return nil
}

func (s *expenseService) calculateBalanceUpdates(expense *repository.Expense, splits []repository.ExpenseSplit) []repository.BalanceUpdate {
	balanceUpdates := make([]repository.BalanceUpdate, 0)
	for _, split := range splits {
		if expense.CreatedBy != split.UserID {
			// Update balance for each user involved in the split relative to the CreatedBy user
			// The net amount represents how much the split.UserID owes the expense.CreatedBy user
			// A positive net amount means split.UserID owes CreatedBy
			// A negative net amount means CreatedBy owes split.UserID
			netAmountOwedToCreator := split.AmountOwed - split.AmountPaid

			if netAmountOwedToCreator != 0 {
				balanceUpdates = append(balanceUpdates, repository.BalanceUpdate{
					User1ID: expense.CreatedBy,
					User2ID: split.UserID,
					Amount:  netAmountOwedToCreator,
				})
			}
		}
	}
	return balanceUpdates
}

func (s *expenseService) CreateExpense(req CreateExpenseRequest) (*repository.Expense, error) {
	if err := s.resolveUserEmailsToIDs(&req); err != nil {
		return nil, err
	}

	expense := &repository.Expense{
		Description: req.Description,
		Tag:         req.Tag,
		TotalAmount: req.TotalAmount,
		CreatedBy:   req.CreatedByID, // Use the resolved ID
	}

	splits, err := s.calculateExpenseSplits(req) // No longer passing usersMap
	if err != nil {
		return nil, err
	}

	// The total amount paid across all splits should match the TotalAmount of the expense
	var totalAmountPaidInSplits float64
	for _, split := range splits {
		totalAmountPaidInSplits += split.AmountPaid
	}

	if roundToTwoDecimalPlaces(totalAmountPaidInSplits) != roundToTwoDecimalPlaces(req.TotalAmount) {
		return nil, fmt.Errorf("total amount paid across all splits (%.2f) does not match total expense amount (%.2f)", totalAmountPaidInSplits, req.TotalAmount)
	}

	// Calculate balance updates
	balanceUpdates := s.calculateBalanceUpdates(expense, splits)

	createdExpense, err := s.expenseRepo.CreateExpense(expense, splits, balanceUpdates)
	if err != nil {
		return nil, fmt.Errorf("failed to create expense in service: %w", err)
	}

	return createdExpense, nil
}

func (s *expenseService) GetExpensesForUser(userEmail string) ([]repository.UserExpenseView, error) {
	users, err := s.userService.GetUsersByEmails([]string{userEmail})
	if err != nil || len(users) == 0 {
		return nil, fmt.Errorf("user with email %s not found", userEmail)
	}

	userID := users[0].ID
	expenses, err := s.expenseRepo.GetExpensesByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get expenses for user %s: %w", userEmail, err)
	}

	return expenses, nil
}
