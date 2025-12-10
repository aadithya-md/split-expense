package repository

import (
	"database/sql"
	"fmt"
	"time"
)

type Balance struct {
	User1ID     int       `json:"user1_id"`
	User2ID     int       `json:"user2_id"`
	Balance     float64   `json:"balance"`
	LastUpdated time.Time `json:"last_updated"`
}

type BalanceRepository interface {
	UpdateBalance(tx *sql.Tx, user1ID, user2ID int, amount float64) error
}

type balanceRepository struct {
	db *sql.DB
}

func NewBalanceRepository(db *sql.DB) BalanceRepository {
	return &balanceRepository{db: db}
}

func (r *balanceRepository) UpdateBalance(tx *sql.Tx, user1ID, user2ID int, amount float64) error {
	// Ensure user1ID is always less than user2ID for consistent keying
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
		amount = -amount // Reverse amount if IDs are swapped
	}

	query := `
		INSERT INTO balances (user1_id, user2_id, balance, last_updated)
		VALUES (?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE
		balance = balance + ?, last_updated = NOW()
	`

	_, err := tx.Exec(query, user1ID, user2ID, amount, amount)
	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	return nil
}
