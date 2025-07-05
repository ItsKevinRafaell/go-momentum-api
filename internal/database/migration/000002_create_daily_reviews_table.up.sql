-- Membuat tabel untuk menyimpan riwayat review harian
CREATE TABLE daily_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    review_date DATE NOT NULL,
    summary_json JSONB NOT NULL, -- Untuk menyimpan statistik (completed, missed, dll)
    ai_feedback_text TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, review_date) -- Satu user hanya bisa punya satu review per hari
);

CREATE INDEX idx_daily_reviews_user_id_date ON daily_reviews(user_id, review_date);