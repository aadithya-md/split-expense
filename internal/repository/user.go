package repository

import (
	"database/sql"
	"fmt"
	"strings"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UserRepository interface {
	CreateUser(user *User) (*User, error)
	GetUser(id int) (*User, error)
	GetUsersByEmails(emails []string) ([]*User, error)
	GetUsersByIDs(ids []int) ([]*User, error)
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CreateUser(user *User) (*User, error) {
	query := "INSERT INTO users (name, email) VALUES (?, ?)"
	result, err := r.db.Exec(query, user.Name, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	user.ID = int(id)
	return user, nil
}

func (r *userRepository) GetUser(id int) (*User, error) {
	query := "SELECT id, name, email FROM users WHERE id = ?"
	user := &User{}
	err := r.db.QueryRow(query, id).Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (r *userRepository) GetUsersByEmails(emails []string) ([]*User, error) {
	if len(emails) == 0 {
		return []*User{}, nil
	}

	placeholders := make([]string, len(emails))
	args := make([]interface{}, len(emails))
	for i, email := range emails {
		placeholders[i] = "?"
		args[i] = email
	}

	query := fmt.Sprintf("SELECT id, name, email FROM users WHERE email IN (%s)", strings.Join(placeholders, ", "))
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by emails: %w", err)
	}
	defer rows.Close()

	var users []*User
	foundEmails := make(map[string]bool)
	for rows.Next() {
		user := &User{}
		if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
		foundEmails[user.Email] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	// Check if all requested emails were found
	if len(users) != len(emails) {
		missingEmails := []string{}
		for _, email := range emails {
			if !foundEmails[email] {
				missingEmails = append(missingEmails, email)
			}
		}
		return nil, fmt.Errorf("some users not found for emails: %s", strings.Join(missingEmails, ", "))
	}

	return users, nil
}

func (r *userRepository) GetUsersByIDs(ids []int) ([]*User, error) {
	if len(ids) == 0 {
		return []*User{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf("SELECT id, name, email FROM users WHERE id IN (%s)", strings.Join(placeholders, ", "))
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by IDs: %w", err)
	}
	defer rows.Close()

	var users []*User
	foundIDs := make(map[int]bool)
	for rows.Next() {
		user := &User{}
		if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
		foundIDs[user.ID] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	// Check if all requested IDs were found
	if len(users) != len(ids) {
		missingIDs := []string{}
		for _, id := range ids {
			if !foundIDs[id] {
				missingIDs = append(missingIDs, fmt.Sprintf("%d", id))
			}
		}
		return nil, fmt.Errorf("some users not found for IDs: %s", strings.Join(missingIDs, ", "))
	}

	return users, nil
}
