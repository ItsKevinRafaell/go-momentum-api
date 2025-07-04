// file: internal/repository/review_repository.go
package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DailyReview struct {
	UserID      string
	ReviewDate  time.Time
	Summary     []TaskSummary // Kita bisa gunakan lagi struct dari task_repository
	AIFeedback  string
}

type ReviewRepository struct {
	db *pgxpool.Pool
}

func NewReviewRepository(db *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{db: db}
}

// CreateOrUpdateReview akan menyimpan atau memperbarui review untuk tanggal tertentu.
func (r *ReviewRepository) CreateOrUpdateReview(ctx context.Context, review *DailyReview) error {
	summaryJSON, err := json.Marshal(review.Summary)
	if err != nil {
		return err
	}

	// Menggunakan ON CONFLICT untuk operasi "UPSERT" (Update atau Insert)
	// Jika data untuk user_id dan review_date sudah ada, ia akan UPDATE. Jika tidak, ia akan INSERT.
	sql := `
		INSERT INTO daily_reviews (user_id, review_date, summary_json, ai_feedback_text)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, review_date)
		DO UPDATE SET summary_json = EXCLUDED.summary_json, ai_feedback_text = EXCLUDED.ai_feedback_text
	`

	_, err = r.db.Exec(ctx, sql, review.UserID, review.ReviewDate, summaryJSON, review.AIFeedback)
	return err
}

func (r *ReviewRepository) GetReviewByDate(ctx context.Context, userID string, reviewDate time.Time) (*DailyReview, error) {
	var review DailyReview
    var summaryJSON []byte // Tampung JSON sebagai byte slice

	sql := `SELECT user_id, review_date, summary_json, ai_feedback_text
	        FROM daily_reviews 
	        WHERE user_id = $1 AND review_date = $2`

	err := r.db.QueryRow(ctx, sql, userID, reviewDate).Scan(
        &review.UserID,
        &review.ReviewDate,
        &summaryJSON,
        &review.AIFeedback,
    )
	if err != nil {
		return nil, err
	}

    // Unmarshal data JSON ke dalam struct
    if err := json.Unmarshal(summaryJSON, &review.Summary); err != nil {
        return nil, err
    }

	return &review, nil
}