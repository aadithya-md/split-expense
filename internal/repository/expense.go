package repository

import (
	"database/sql"
	"fmt"
	"time"
)

type Expense struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	TotalAmount float64   `json:"total_amount"`
	CreatedBy   int       `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type ExpenseSplit struct {
	ID         int     `json:"id"`
	ExpenseID  int     `json:"expense_id"`
	UserID     int     `json:"user_id"`
	AmountPaid float64 `json:"amount_paid"`
	AmountOwed float64 `json:"amount_owed"`
}

type ExpenseRepository interface {
	CreateExpense(expense *Expense, splits []ExpenseSplit) (*Expense, error)
}

type expenseRepository struct {
	db          *sql.DB
	balanceRepo BalanceRepository
}

func NewExpenseRepository(db *sql.DB, balanceRepo BalanceRepository) ExpenseRepository {
	return &expenseRepository{db: db, balanceRepo: balanceRepo}
}

func (r *expenseRepository) CreateExpense(expense *Expense, splits []ExpenseSplit) (*Expense, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on error, no-op on commit

	// Insert expense
	expenseQuery := "INSERT INTO expenses (description, total_amount, created_by, created_at) VALUES (?, ?, ?, ?)"
	expense.CreatedAt = time.Now() // Set CreatedAt before insertion
	result, err := tx.Exec(expenseQuery, expense.Description, expense.TotalAmount, expense.CreatedBy, expense.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create expense: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID for expense: %w", err)
	}
	expense.ID = int(id)

	// Insert expense splits and update balances
	for _, split := range splits {
		// Insert split
		splitQuery := "INSERT INTO expense_splits (expense_id, user_id, amount_paid, amount_owed) VALUES (?, ?, ?, ?)"
		_, err := tx.Exec(splitQuery, expense.ID, split.UserID, split.AmountPaid, split.AmountOwed)
		if err != nil {
			return nil, fmt.Errorf("failed to create expense split: %w", err)
		}

		// Only update balance if the split is between two different users
		if expense.CreatedBy != split.UserID {
			// Update balance for each user involved in the split relative to the CreatedBy user
			// The net amount represents how much the split.UserID owes the expense.CreatedBy user
			// A positive net amount means split.UserID owes CreatedBy
			// A negative net amount means CreatedBy owes split.UserID
			netAmountOwedToCreator := split.AmountOwed - split.AmountPaid

			if netAmountOwedToCreator != 0 {
				err = r.balanceRepo.UpdateBalance(tx, expense.CreatedBy, split.UserID, netAmountOwedToCreator)
				if err != nil {
					return nil, fmt.Errorf("failed to update balance for user %d: %w", split.UserID, err)
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return expense, nil
}
