package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Goal struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

type GoalRepository struct {
	db *pgxpool.Pool
}

func NewGoalRepository(db *pgxpool.Pool) *GoalRepository {
	return &GoalRepository{db: db}
}

// CreateGoal menyimpan goal baru ke database dan mengembalikan ID-nya.
func (r *GoalRepository) CreateGoal(ctx context.Context, goal *Goal) (string, error) {
	var id string
	sql := "INSERT INTO goals (user_id, description, is_active) VALUES ($1, $2, $3) RETURNING id"
	err := r.db.QueryRow(ctx, sql, goal.UserID, goal.Description, goal.IsActive).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *GoalRepository) GetActiveGoalByUserID(ctx context.Context, userID string) (*Goal, error) {
	var goal Goal
	sql := "SELECT id, user_id, description, is_active FROM goals WHERE user_id = $1 AND is_active = TRUE LIMIT 1"
	err := r.db.QueryRow(ctx, sql, userID).Scan(&goal.ID, &goal.UserID, &goal.Description, &goal.IsActive)
	if err != nil {
		return nil, err // Akan mengembalikan error jika tidak ada baris yang ditemukan
	}
	return &goal, nil
}