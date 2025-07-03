package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoadmapStep struct {
	ID      string `json:"id"`
	GoalID  string `json:"goal_id"`
	Order   int    `json:"step_order"`
	Title   string `json:"title"`
	Status  string `json:"status"`
}

type RoadmapRepository struct {
	db *pgxpool.Pool
}

func NewRoadmapRepository(db *pgxpool.Pool) *RoadmapRepository {
	return &RoadmapRepository{db: db}
}

// CreateRoadmapSteps memasukkan beberapa langkah roadmap sekaligus.
func (r *RoadmapRepository) CreateRoadmapSteps(ctx context.Context, steps []RoadmapStep) error {
	// Kita akan menggunakan fitur CopyFrom dari pgx untuk bulk insert yang efisien.
	rows := make([][]interface{}, len(steps))
	for i, step := range steps {
		rows[i] = []interface{}{step.GoalID, step.Order, step.Title, step.Status}
	}

	_, err := r.db.CopyFrom(
		ctx,
		pgx.Identifier{"roadmap_steps"},
		[]string{"goal_id", "step_order", "title", "status"},
		pgx.CopyFromRows(rows),
	)

	return err
}

func (r *RoadmapRepository) GetRoadmapStepsByGoalID(ctx context.Context, goalID string) ([]RoadmapStep, error) {
	var steps []RoadmapStep
	sql := "SELECT id, goal_id, step_order, title, status FROM roadmap_steps WHERE goal_id = $1 ORDER BY step_order ASC"
	rows, err := r.db.Query(ctx, sql, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var step RoadmapStep
		if err := rows.Scan(&step.ID, &step.GoalID, &step.Order, &step.Title, &step.Status); err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}

	return steps, nil
}

// DeleteRoadmapStepsByGoalID menghapus semua langkah roadmap yang terhubung ke sebuah goal.
func (r *RoadmapRepository) DeleteRoadmapStepsByGoalID(ctx context.Context, goalID string) error {
    sql := "DELETE FROM roadmap_steps WHERE goal_id = $1"
    _, err := r.db.Exec(ctx, sql, goalID)
    return err
}

func (r *RoadmapRepository) GetLastStepOrder(ctx context.Context, goalID string) (int, error) {
    var lastOrder int
    sql := "SELECT COALESCE(MAX(step_order), 0) FROM roadmap_steps WHERE goal_id = $1"
    err := r.db.QueryRow(ctx, sql, goalID).Scan(&lastOrder)
    if err != nil {
        return 0, err
    }
    return lastOrder, nil
}

// CreateRoadmapStep menyimpan satu langkah roadmap baru.
func (r *RoadmapRepository) CreateRoadmapStep(ctx context.Context, step *RoadmapStep) (*RoadmapStep, error) {
    var createdStep RoadmapStep
    sql := `INSERT INTO roadmap_steps (goal_id, step_order, title, status)
            VALUES ($1, $2, $3, $4)
            RETURNING id, goal_id, step_order, title, status`

    err := r.db.QueryRow(ctx, sql, step.GoalID, step.Order, step.Title, step.Status).Scan(
        &createdStep.ID,
        &createdStep.GoalID,
        &createdStep.Order,
        &createdStep.Title,
        &createdStep.Status,
    )
    if err != nil {
        return nil, err
    }
    return &createdStep, nil
}