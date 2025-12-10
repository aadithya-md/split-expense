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
	GetBalancesByUserID(userID int) ([]Balance, error)
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

func (r *balanceRepository) GetBalancesByUserID(userID int) ([]Balance, error) {
	query := `
		SELECT user1_id, user2_id, balance, last_updated
		FROM balances
		WHERE user1_id = ? OR user2_id = ?
		ORDER BY last_updated DESC
	`

	rows, err := r.db.Query(query, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query balances for user %d: %w", userID, err)
	}
	defer rows.Close()

	var balances []Balance
	for rows.Next() {
		var b Balance
		if err := rows.Scan(&b.User1ID, &b.User2ID, &b.Balance, &b.LastUpdated); err != nil {
			return nil, fmt.Errorf("failed to scan balance row for user %d: %w", userID, err)
		}
		balances = append(balances, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over balance rows for user %d: %w", userID, err)
	}

	return balances, nil
}
