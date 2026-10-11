package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Laytey/gohospital/appointment-service/internal/db"
	"github.com/Laytey/gohospital/appointment-service/internal/models"
	"github.com/google/uuid"
)

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

	err := r.storage.Pool.QueryRow(
		ctx,
		query,
		appointment.ID,
		appointment.PatientID,
		appointment.Doctor,
		appointment.Time,
		appointment.Status,
	).Scan(&appointment.CreatedAt)
	return err
}

func (r *PostgresAppointmentRepository) GetByID(ctx context.Context, id string) (models.Appointment, error) {
	query := `SELECT id, patient_id, doctor, time, status, created_at
	FROM appointments
	WHERE id = $1`

	var a models.Appointment
	err := r.storage.Pool.QueryRow(ctx, query, id).Scan(&a.ID, &a.PatientID, &a.Doctor, &a.Time, &a.Status, &a.CreatedAt)
	if err != nil {
		return models.Appointment{}, fmt.Errorf("failed to get appointment by ID: %w", err)
	}
	return a, nil
}

func (r *PostgresAppointmentRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE appointments
	SET status = $1
	WHERE id = $2`

	result, err := r.storage.Pool.Exec(ctx, query, status, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		// RowsAffected — сколько СТРОК (записей таблицы) обновлено
		return errors.New("appointment not found")
	}
	return nil
}

func (r *PostgresAppointmentRepository) ListByPatient(ctx context.Context, patientID string) ([]models.Appointment, error) {
	query := `SELECT id, patient_id, doctor, time, status, created_at
	FROM appointments
	WHERE patient_id = $1
	ORDER BY time DESC`

	rows, err := r.storage.Pool.Query(ctx, query, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Appointment
	for rows.Next() {
		var a models.Appointment
		if err := rows.Scan(&a.ID, &a.PatientID, &a.Doctor, &a.Time, &a.Status, &a.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, nil

}
