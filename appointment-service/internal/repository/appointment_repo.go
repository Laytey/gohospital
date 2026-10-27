package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Laytey/gohospital/appointment-service/internal/db"
	"github.com/Laytey/gohospital/appointment-service/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrAppointmentNotFound возвращается, когда записи с указанным ID нет в БД.
// Это sentinel-ошибка: вызывающий код должен проверять её через errors.Is().
var ErrAppointmentNotFound = errors.New("appointment not found")

// AppointmentRepository определяет контракт для работы с записями.
type AppointmentRepository interface {
	// Save сохраняет новую запись и возвращает её с присвоенным ID.
	Save(ctx context.Context, a *models.Appointment) error

	// GetByID возвращает запись по UUID.
	// Возвращает ошибку "appointment not found", если записи нет.
	GetByID(ctx context.Context, id string) (models.Appointment, error)

	// UpdateStatus обновляет статус записи ("created" / "canceled").
	// Возвращает ошибку, если записи с таким ID нет.
	UpdateStatus(ctx context.Context, id string, status string) error

	// ListByPatient возвращает все записи указанного пациента.
	ListByPatient(ctx context.Context, patientID string) ([]models.Appointment, error)
}

type PostgresAppointmentRepository struct {
	storage *db.Storage
}

func NewPostgresAppointmentRepository(storage *db.Storage) *PostgresAppointmentRepository {
	return &PostgresAppointmentRepository{
		storage: storage,
	}
}

func (r *PostgresAppointmentRepository) Save(ctx context.Context, appointment *models.Appointment) error {
	appointment.ID = uuid.NewString()

	query := `INSERT INTO appointments (id, patient_id, doctor, time, status)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING created_at`

	if err := r.storage.Pool.QueryRow(
		ctx,
		query,
		appointment.ID,
		appointment.PatientID,
		appointment.Doctor,
		appointment.Time,
		appointment.Status,
	).Scan(&appointment.CreatedAt); err != nil {
		return fmt.Errorf("failed to save appointment: %w", err)
	}
	return nil

}

// GetByID возвращает запись или ErrAppointmentNotFound.
func (r *PostgresAppointmentRepository) GetByID(ctx context.Context, id string) (models.Appointment, error) {
	query := `SELECT id, patient_id, doctor, time, status, created_at
	FROM appointments
	WHERE id = $1`

	var a models.Appointment
	err := r.storage.Pool.QueryRow(ctx, query, id).Scan(&a.ID, &a.PatientID, &a.Doctor, &a.Time, &a.Status, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Appointment{}, ErrAppointmentNotFound
		}
		return models.Appointment{}, fmt.Errorf("failed to get appointment by ID: %w", err)
	}
	return a, nil
}

// UpdateStatus обновляет статус или возвращает ErrAppointmentNotFound.
func (r *PostgresAppointmentRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE appointments
	SET status = $1
	WHERE id = $2`

	result, err := r.storage.Pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update appointment status: %w", err)
	}
	if result.RowsAffected() == 0 {
		// RowsAffected — сколько СТРОК (записей таблицы) обновлено
		return ErrAppointmentNotFound
	}
	return nil
}

// ListByPatient возвращает все записи пациента.
func (r *PostgresAppointmentRepository) ListByPatient(ctx context.Context, patientID string) ([]models.Appointment, error) {
	query := `SELECT id, patient_id, doctor, time, status, created_at
	FROM appointments
	WHERE patient_id = $1
	ORDER BY time DESC`

	rows, err := r.storage.Pool.Query(ctx, query, patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to list appointments: %w", err)
	}
	defer rows.Close()

	var result []models.Appointment
	for rows.Next() {
		var a models.Appointment
		if err := rows.Scan(&a.ID, &a.PatientID, &a.Doctor, &a.Time, &a.Status, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan appointment row: %w", err)
		}
		result = append(result, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return result, nil

}
