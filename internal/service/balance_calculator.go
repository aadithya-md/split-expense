package service

import "github.com/aadithya-md/split-expense/internal/repository"

// CalculateBalanceUpdates determines the balance adjustments needed based on an expense and its splits.
// It returns a slice of BalanceUpdate structs, representing who owes whom.
// BalanceCalculationStrategy defines the interface for different balance calculation methods.
type BalanceCalculationStrategy interface {
	Calculate(expense *repository.Expense, splits []repository.ExpenseSplit) []repository.BalanceUpdate
}

// SimpleBalanceCalculationStrategy provides a concrete implementation of BalanceCalculationStrategy.
type SimpleBalanceCalculationStrategy struct{}

// Calculate determines the balance adjustments needed based on an expense and its splits
// using a simple direct calculation method.
func (s *SimpleBalanceCalculationStrategy) Calculate(expense *repository.Expense, splits []repository.ExpenseSplit) []repository.BalanceUpdate {
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

// getSettlementStrategy returns the appropriate balance calculation strategy.
// For now, it always returns the HighestPositiveBalanceStrategy.
func getSettlementStrategy() BalanceCalculationStrategy {
	return &HighestPositiveBalanceStrategy{}
}

// CalculateBalanceUpdates orchestrates the balance calculation using the internally selected strategy.
func CalculateBalanceUpdates(expense *repository.Expense, splits []repository.ExpenseSplit) []repository.BalanceUpdate {
	strategy := getSettlementStrategy()
	return strategy.Calculate(expense, splits)
}

// HighestPositiveBalanceStrategy defines a strategy where everyone pays to the user with the highest positive balance.
type HighestPositiveBalanceStrategy struct{}

// Calculate calculates balance updates where everyone pays to the user with the highest positive balance.
func (s *HighestPositiveBalanceStrategy) Calculate(expense *repository.Expense, splits []repository.ExpenseSplit) []repository.BalanceUpdate {
	userBalances := make(map[int]float64) // userID -> net balance for this expense

	var highestBalanceUserID int = -1
	highestBalanceAmount := 0.0
	// Initialize balances for all users in the splits
	for _, split := range splits {
		balance := split.AmountPaid - split.AmountOwed
		userBalances[(split.UserID)] = balance
		if balance > highestBalanceAmount {
			highestBalanceAmount = balance
			highestBalanceUserID = split.UserID
		}
	}

	balanceUpdates := make([]repository.BalanceUpdate, 0)
	if highestBalanceUserID == -1 {
		// No one has a positive balance, or expense amount was 0.
		return balanceUpdates
	}

	for userID, balance := range userBalances {
		if userID != (highestBalanceUserID) && balance < 0 {
			// This user owes money to the highest balance user
			balanceUpdates = append(balanceUpdates, repository.BalanceUpdate{
				User1ID: (highestBalanceUserID), // The receiver
				User2ID: (userID),               // The payer
				Amount:  -balance,               // Amount owed is the negative of their balance
			})
		}
	}

	return balanceUpdates
}
