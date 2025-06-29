package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"-"` // Jangan pernah kirim password ke JSON
}

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(ctx context.Context, user *User) (string, error) {
	var id string
	sql := "INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id"
	err := r.db.QueryRow(ctx, sql, user.Email, user.Password).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// GetUserByEmail mencari pengguna berdasarkan alamat email.
// Penting untuk mengembalikan hash password agar bisa diverifikasi di service.
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	sql := "SELECT id, email, password_hash FROM users WHERE email = $1"
	err := r.db.QueryRow(ctx, sql, email).Scan(&user.ID, &user.Email, &user.Password)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, userID string) (*User, error) {
    var user User
    sql := "SELECT id, email, password_hash FROM users WHERE id = $1"
    err := r.db.QueryRow(ctx, sql, userID).Scan(&user.ID, &user.Email, &user.Password)
    if err != nil {
        return nil, err
    }
    return &user, nil
}

// UpdatePasswordHash hanya mengupdate kolom password_hash.
func (r *UserRepository) UpdatePasswordHash(ctx context.Context, userID, newHashedPassword string) error {
    sql := "UPDATE users SET password_hash = $1 WHERE id = $2"
    result, err := r.db.Exec(ctx, sql, newHashedPassword, userID)
    if err != nil {
        return err
    }
    if result.RowsAffected() == 0 {
        return pgx.ErrNoRows
    }
    return nil
}