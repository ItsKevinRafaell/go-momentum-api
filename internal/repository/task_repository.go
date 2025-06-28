package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Kita gunakan lagi struct Task yang sudah pernah kita definisikan di ERD
type Task struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	RoadmapStepID *string    `json:"roadmap_step_id"` // Pointer agar bisa null
	Title         string     `json:"title"`
	Status        string     `json:"status"`
	ScheduledDate time.Time  `json:"scheduled_date"`
	Deadline      *time.Time `json:"deadline"`       // Pointer agar bisa null
	CompletedAt   *time.Time `json:"completed_at"`   // Pointer agar bisa null
}

type TaskSummary struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type TaskRepository struct {
	db *pgxpool.Pool
}

func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: db}
}

// GetTasksByDate mengambil semua tugas untuk user tertentu pada tanggal tertentu.
func (r *TaskRepository) GetTasksByDate(ctx context.Context, userID string, date time.Time) ([]Task, error) {
	var tasks []Task
	// Menggunakan DATE() untuk membandingkan hanya bagian tanggal dari timestamp
	sql := "SELECT id, user_id, roadmap_step_id, title, status, scheduled_date, deadline, completed_at FROM tasks WHERE user_id = $1 AND scheduled_date = $2 ORDER BY created_at ASC"
	rows, err := r.db.Query(ctx, sql, userID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.UserID, &task.RoadmapStepID, &task.Title, &task.Status, &task.ScheduledDate, &task.Deadline, &task.CompletedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// CreateTasks memasukkan beberapa tugas sekaligus menggunakan CopyFrom.
func (r *TaskRepository) CreateTasks(ctx context.Context, tasks []Task) error {
	rows := make([][]interface{}, len(tasks))
	for i, task := range tasks {
		rows[i] = []interface{}{task.UserID, task.RoadmapStepID, task.Title, task.Status, task.ScheduledDate}
	}

	_, err := r.db.CopyFrom(
		ctx,
		pgx.Identifier{"tasks"},
		[]string{"user_id", "roadmap_step_id", "title", "status", "scheduled_date"},
		pgx.CopyFromRows(rows),
	)

	return err
}

func (r *TaskRepository) UpdateTaskStatus(ctx context.Context, userID, taskID, status string) error {
    now := time.Now()
    sql := "UPDATE tasks SET status = $1, completed_at = $2 WHERE id = $3 AND user_id = $4"

    result, err := r.db.Exec(ctx, sql, status, now, taskID, userID)
    if err != nil {
        return err
    }

    // --- PENAMBAHAN LOGIKA PENTING ---
    if result.RowsAffected() == 0 {
        // Jika tidak ada baris yang terpengaruh, artinya taskID tidak ditemukan
        // atau bukan milik userID ini. Kita kembalikan error standar "no rows".
        return pgx.ErrNoRows 
    }

    return nil
}

// UpdateTaskDeadline memperbarui batas waktu untuk sebuah tugas.
func (r *TaskRepository) UpdateTaskDeadline(ctx context.Context, userID, taskID string, deadline time.Time) error {
    sql := "UPDATE tasks SET deadline = $1 WHERE id = $2 AND user_id = $3"
    result, err := r.db.Exec(ctx, sql, deadline, taskID, userID)
    if err != nil {
        return err
    }
    if result.RowsAffected() == 0 {
        return pgx.ErrNoRows
    }
    return nil
}

func (r *TaskRepository) UpdateTaskTitle(ctx context.Context, userID, taskID, title string) error {
    sql := "UPDATE tasks SET title = $1 WHERE id = $2 AND user_id = $3"
    result, err := r.db.Exec(ctx, sql, title, taskID, userID)
    if err != nil {
        return err
    }
    if result.RowsAffected() == 0 {
        return pgx.ErrNoRows
    }
    return nil
}

func (r *TaskRepository) CreateTask(ctx context.Context, task *Task) (*Task, error) {
	var createdTask Task
	sql := `INSERT INTO tasks (user_id, title, status, scheduled_date, deadline) 
	        VALUES ($1, $2, $3, $4, $5) 
	        RETURNING id, user_id, roadmap_step_id, title, status, scheduled_date, deadline, completed_at`

	err := r.db.QueryRow(ctx, sql, task.UserID, task.Title, task.Status, task.ScheduledDate, task.Deadline).Scan(
		&createdTask.ID,
		&createdTask.UserID,
		&createdTask.RoadmapStepID,
		&createdTask.Title,
		&createdTask.Status,
		&createdTask.ScheduledDate,
		&createdTask.Deadline,
		&createdTask.CompletedAt,
	)

	if err != nil {
		return nil, err
	}
	return &createdTask, nil
}

func (r *TaskRepository) DeleteTask(ctx context.Context, userID, taskID string) error {
	sql := "DELETE FROM tasks WHERE id = $1 AND user_id = $2"

	// Sekali lagi, AND user_id = $2 adalah penjaga keamanan kita.
	result, err := r.db.Exec(ctx, sql, taskID, userID)
	if err != nil {
		return err
	}

    // Opsional: periksa apakah ada baris yang benar-benar terhapus.
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows // Menggunakan error standar jika tidak ada yang terhapus
	}

	return nil
}

func (r *TaskRepository) FinalizeMissedTasks(ctx context.Context, userID string, date time.Time) error {
	// Hanya update tugas untuk tanggal yang dijadwalkan, yang statusnya masih pending,
	// dan yang deadline-nya sudah lewat dari waktu sekarang.
	sql := `UPDATE tasks SET status = 'missed' 
	        WHERE user_id = $1 AND scheduled_date = $2 AND status = 'pending' AND deadline < NOW()`

	_, err := r.db.Exec(ctx, sql, userID, date)
	return err
}

func (r *TaskRepository) GetTaskSummaryByDate(ctx context.Context, userID string, date time.Time) ([]TaskSummary, error) {
	var summaries []TaskSummary
	sql := `SELECT status, COUNT(*) as count 
	        FROM tasks 
	        WHERE user_id = $1 AND scheduled_date = $2 
	        GROUP BY status`

	rows, err := r.db.Query(ctx, sql, userID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var summary TaskSummary
		if err := rows.Scan(&summary.Status, &summary.Count); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}