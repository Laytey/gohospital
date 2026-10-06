package repository

import (
	"context"
	"errors"

	"github.com/Laytey/gohospital/patient-service/internal/db"
	"github.com/Laytey/gohospital/patient-service/internal/models"
	"github.com/google/uuid"
)

// PatientRepository определяет контракт для работы с пациентами.
type PatientRepository interface {
	// Save сохраняет нового пациента и возвращает его с присвоенным ID.
	Save(ctx context.Context, patient *models.Patient) error

	// GetByID возвращает пациента по UUID.
	// Возвращает ошибку "patient not found", если пациента нет.
	GetByID(ctx context.Context, id string) (models.Patient, error)
}

// PostgresPatientRepository — реализация PatientRepository на PostgreSQL.
type PostgresPatientRepository struct {
	storage *db.Storage
}

// NewPostgresPatientRepository создаёт репозиторий с указанным пулом соединений.
func NewPostgresPatientRepository(storage *db.Storage) *PostgresPatientRepository {
	return &PostgresPatientRepository{
		storage: storage,
	}
}

// Save реализует PatientRepository.Save.
func (r *PostgresPatientRepository) Save(ctx context.Context, patient *models.Patient) error {
	patient.ID = uuid.NewString()

	query := `INSERT INTO patients (id, name, birthdate) 
	VALUES ($1, $2, $3)
	RETURNING created_at`

	err := r.storage.Pool.QueryRow(ctx, query, patient.ID, patient.Name, patient.Birthdate).Scan(&patient.CreatedAt)
	return err
}

// GetByID реализует PatientRepository.GetByID.
func (r *PostgresPatientRepository) GetByID(ctx context.Context, id string) (models.Patient, error) {
	query := `SELECT id, name, birthdate, created_at 
	FROM patients 
	WHERE id = $1`

	var p models.Patient
	err := r.storage.Pool.QueryRow(ctx, query, id).Scan(&p.ID, &p.Name, &p.Birthdate, &p.CreatedAt)
	if err != nil {
		return models.Patient{}, errors.New("patient not found")
	}
	return p, nil
}
