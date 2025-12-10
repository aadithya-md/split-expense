package repository

import (
	"database/sql"
	"fmt"
	"time"
)

type Expense struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	Tag         string    `json:"tag"`
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

type BalanceUpdate struct {
	User1ID int
	User2ID int
	Amount  float64
}

type UserExpenseView struct {
	Date        time.Time `json:"date"`
	Tag         string    `json:"tag"`
	Description string    `json:"description"`
	TotalAmount float64   `json:"total_amount"`
	Share       float64   `json:"share"`
}

type ExpenseRepository interface {
	CreateExpense(expense *Expense, splits []ExpenseSplit, balanceUpdates []BalanceUpdate) (*Expense, error)
	GetExpensesByUserID(userID int) ([]UserExpenseView, error)
}

type expenseRepository struct {
	db          *sql.DB
	balanceRepo BalanceRepository
}

func NewExpenseRepository(db *sql.DB, balanceRepo BalanceRepository) ExpenseRepository {
	return &expenseRepository{db: db, balanceRepo: balanceRepo}
}

func (r *expenseRepository) CreateExpense(expense *Expense, splits []ExpenseSplit, balanceUpdates []BalanceUpdate) (*Expense, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on error, no-op on commit

	// Insert expense
	expenseQuery := "INSERT INTO expenses (description, tag, total_amount, created_by, created_at) VALUES (?, ?, ?, ?, ?)"
	expense.CreatedAt = time.Now() // Set CreatedAt before insertion
	result, err := tx.Exec(expenseQuery, expense.Description, expense.Tag, expense.TotalAmount, expense.CreatedBy, expense.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create expense: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID for expense: %w", err)
	}
	expense.ID = int(id)

	// Insert expense splits
	for _, split := range splits {
		// Insert split
		splitQuery := "INSERT INTO expense_splits (expense_id, user_id, amount_paid, amount_owed) VALUES (?, ?, ?, ?)"
		_, err := tx.Exec(splitQuery, expense.ID, split.UserID, split.AmountPaid, split.AmountOwed)
		if err != nil {
			return nil, fmt.Errorf("failed to create expense split: %w", err)
		}
	}

	// Update balances
	for _, update := range balanceUpdates {
		err = r.balanceRepo.UpdateBalance(tx, update.User1ID, update.User2ID, update.Amount)
		if err != nil {
			return nil, fmt.Errorf("failed to update balance between user %d and %d: %w", update.User1ID, update.User2ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return expense, nil
}

func (r *expenseRepository) GetExpensesByUserID(userID int) ([]UserExpenseView, error) {
	query := `
		SELECT
			e.created_at,
			e.tag,
			e.description,
			e.total_amount,
			es.amount_paid,
			es.amount_owed
		FROM
			expenses e
		JOIN
			expense_splits es ON e.id = es.expense_id
		WHERE
			es.user_id = ?
		ORDER BY
			e.created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query expenses for user %d: %w", userID, err)
	}
	defer rows.Close()

	var expenses []UserExpenseView
	for rows.Next() {
		var (
			Date        time.Time
			Tag         string
			Description string
			TotalAmount float64
			AmountPaid  float64
			AmountOwed  float64
		)

		if err := rows.Scan(&Date, &Tag, &Description, &TotalAmount, &AmountPaid, &AmountOwed); err != nil {
			return nil, fmt.Errorf("failed to scan expense row for user %d: %w", userID, err)
		}

		expenses = append(expenses, UserExpenseView{
			Date:        Date,
			Tag:         Tag,
			Description: Description,
			TotalAmount: TotalAmount,
			Share:       AmountPaid - AmountOwed,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over expense rows for user %d: %w", userID, err)
	}

	return expenses, nil
}
