package service

import (
	"fmt"
	"math"

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
}

type expenseService struct {
	expenseRepo repository.ExpenseRepository
	userRepo    repository.UserRepository // Added UserRepository
}

func NewExpenseService(expenseRepo repository.ExpenseRepository, userRepo repository.UserRepository) ExpenseService {
	return &expenseService{expenseRepo: expenseRepo, userRepo: userRepo}
}

// roundToTwoDecimalPlaces rounds a float64 to two decimal places.
func roundToTwoDecimalPlaces(f float64) float64 {
	return math.Round(f*100) / 100
}

type SplitStrategy interface {
	CalculateSplits(req CreateExpenseRequest) ([]repository.ExpenseSplit, error) // Removed usersMap
}

type equalSplitStrategy struct{}

func (s *equalSplitStrategy) CalculateSplits(req CreateExpenseRequest) ([]repository.ExpenseSplit, error) {
	if len(req.EqualSplits) == 0 {
		return nil, fmt.Errorf("equal split requires participants")
	}

	amountPerUser := roundToTwoDecimalPlaces(req.TotalAmount / float64(len(req.EqualSplits)))

	splits := make([]repository.ExpenseSplit, 0, len(req.EqualSplits))
	var currentTotalOwed float64

	for i, es := range req.EqualSplits {
		// UserID is now populated by resolveUserEmailsToIDs
		splitOwed := amountPerUser
		if i == 0 { // Distribute rounding error to the first user
			splitOwed = roundToTwoDecimalPlaces(req.TotalAmount - (amountPerUser * float64(len(req.EqualSplits)-1)))
		}
		splits = append(splits, repository.ExpenseSplit{
			UserID:     es.UserID, // Use pre-populated UserID
			AmountPaid: roundToTwoDecimalPlaces(es.AmountPaid),
			AmountOwed: splitOwed,
		})
		currentTotalOwed += splitOwed
	}

	// Final check to ensure total owed matches total amount after rounding adjustments
	if roundToTwoDecimalPlaces(currentTotalOwed) != roundToTwoDecimalPlaces(req.TotalAmount) {
		return nil, fmt.Errorf("rounding error: sum of equal split amounts (%.2f) does not match total amount (%.2f)", currentTotalOwed, req.TotalAmount)
	}

	return splits, nil
}

type percentageSplitStrategy struct{}

func (s *percentageSplitStrategy) CalculateSplits(req CreateExpenseRequest) ([]repository.ExpenseSplit, error) {
	if len(req.PercentageSplits) == 0 {
		return nil, fmt.Errorf("percentage split requires percentages")
	}

	var totalPercentage float64
	for _, ps := range req.PercentageSplits {
		totalPercentage += ps.Percentage
	}
	if totalPercentage != 100 {
		return nil, fmt.Errorf("percentage split total must be 100%%")
	}

	splits := make([]repository.ExpenseSplit, 0, len(req.PercentageSplits))
	var currentTotalOwed float64

	for _, ps := range req.PercentageSplits {
		// UserID is now populated by resolveUserEmailsToIDs
		splitOwed := roundToTwoDecimalPlaces(req.TotalAmount * (ps.Percentage / 100))
		splits = append(splits, repository.ExpenseSplit{
			UserID:     ps.UserID, // Use pre-populated UserID
			AmountPaid: roundToTwoDecimalPlaces(ps.AmountPaid),
			AmountOwed: splitOwed,
		})
		currentTotalOwed += splitOwed
	}

	// Adjust for rounding errors
	diff := roundToTwoDecimalPlaces(req.TotalAmount - currentTotalOwed)
	if diff != 0 && len(splits) > 0 {
		splits[0].AmountOwed = roundToTwoDecimalPlaces(splits[0].AmountOwed + diff)
	}

	return splits, nil
}

type manualSplitStrategy struct{}

func (s *manualSplitStrategy) CalculateSplits(req CreateExpenseRequest) ([]repository.ExpenseSplit, error) {
	if len(req.ManualSplits) == 0 {
		return nil, fmt.Errorf("manual split requires manual amounts")
	}

	var totalOwed float64
	splits := make([]repository.ExpenseSplit, 0, len(req.ManualSplits))
	for _, ms := range req.ManualSplits {
		// UserID is now populated by resolveUserEmailsToIDs
		splitOwed := roundToTwoDecimalPlaces(ms.AmountOwed)
		splits = append(splits, repository.ExpenseSplit{
			UserID:     ms.UserID, // Use pre-populated UserID
			AmountPaid: roundToTwoDecimalPlaces(ms.AmountPaid),
			AmountOwed: splitOwed,
		})
		totalOwed += splitOwed
	}

	if roundToTwoDecimalPlaces(totalOwed) != roundToTwoDecimalPlaces(req.TotalAmount) {
		return nil, fmt.Errorf("manual split amounts (%.2f) must sum up to total amount (%.2f)", totalOwed, req.TotalAmount)
	}

	return splits, nil
}

func (s *expenseService) getSplitStrategy(method SplitMethodType) (SplitStrategy, error) {
	switch method {
	case SplitMethodEqual:
		return &equalSplitStrategy{}, nil
	case SplitMethodPercentage:
		return &percentageSplitStrategy{}, nil
	case SplitMethodManual:
		return &manualSplitStrategy{}, nil
	default:
		return nil, fmt.Errorf("invalid split method: %s", method)
	}
}

func (s *expenseService) calculateExpenseSplits(req CreateExpenseRequest) ([]repository.ExpenseSplit, error) {
	strategy, err := s.getSplitStrategy(req.SplitMethod)
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
	usersSlice, err := s.userRepo.GetUsersByEmails(emailList)
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

func (s *expenseService) CreateExpense(req CreateExpenseRequest) (*repository.Expense, error) {
	if err := s.resolveUserEmailsToIDs(&req); err != nil {
		return nil, err
	}

	expense := &repository.Expense{
		Description: req.Description,
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

	createdExpense, err := s.expenseRepo.CreateExpense(expense, splits)
	if err != nil {
		return nil, fmt.Errorf("failed to create expense in service: %w", err)
	}

	return createdExpense, nil
}
