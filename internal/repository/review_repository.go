// file: internal/repository/review_repository.go
package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Struct ini sekarang dengan json tag yang eksplisit
type DailyReview struct {
	UserID     string        `json:"userId"`
	ReviewDate time.Time     `json:"reviewDate"`
	Summary    []TaskSummary `json:"summary"`
	AIFeedback string        `json:"aiFeedback"`
}

type ReviewRepository struct {
	db *pgxpool.Pool
}

func NewReviewRepository(db *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) CreateOrUpdateReview(ctx context.Context, review *DailyReview) error {
	summaryJSON, err := json.Marshal(review.Summary)
	if err != nil {
		return err
	}
	sql := `
		INSERT INTO daily_reviews (user_id, review_date, summary_json, ai_feedback_text)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, review_date)
		DO UPDATE SET summary_json = EXCLUDED.summary_json, ai_feedback_text = EXCLUDED.ai_feedback_text`
	_, err = r.db.Exec(ctx, sql, review.UserID, review.ReviewDate, summaryJSON, review.AIFeedback)
	return err
}

func (r *ReviewRepository) GetReviewByDate(ctx context.Context, userID string, reviewDate time.Time) (*DailyReview, error) {
	var review DailyReview
	var summaryJSON []byte
	sql := `SELECT user_id, review_date, summary_json, ai_feedback_text FROM daily_reviews WHERE user_id = $1 AND review_date = $2`
	err := r.db.QueryRow(ctx, sql, userID, reviewDate).Scan(
		&review.UserID,
		&review.ReviewDate,
		&summaryJSON,
		&review.AIFeedback,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(summaryJSON, &review.Summary); err != nil {
		return nil, err
	}
	return &review, nil
}