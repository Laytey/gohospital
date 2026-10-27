package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Laytey/gohospital/appointment-service/internal/db"
	"github.com/Laytey/gohospital/appointment-service/internal/models"
	"github.com/google/uuid"
)

// OutboxRepository — репозиторий для работы с outbox-событиями.
type OutboxRepository interface {
	// SaveEventOnly вставляет только событие в outbox, без записи в appointments.
	// Используется, когда бизнес-сущность уже существует (например, при отмене).
	SaveEventOnly(ctx context.Context, topic string, payload []byte) error

	// SaveWithAppointment сохраняет событие в outbox в одной транзакции с записью.
	// Если запись не сохранится — событие тоже не сохранится (и наоборот).
	SaveInTx(ctx context.Context, a *models.Appointment, topic string, payload []byte) error

	// GetUnsent возвращает до limit неотправленных событий, старейшие первыми.
	GetUnsent(ctx context.Context, limit int) ([]models.OutboxEvent, error)

	// MarkSent помечает событие как отправленное.
	MarkSent(ctx context.Context, id string) error
}

type PostgresOutboxRepository struct {
	storage *db.Storage
}

func NewPostgresOutboxRepository(storage *db.Storage) *PostgresOutboxRepository {
	return &PostgresOutboxRepository{storage: storage}
}

// SaveEventOnly вставляет только событие в outbox (без транзакции с appointment).
func (r *PostgresOutboxRepository) SaveEventOnly(ctx context.Context, topic string, payload []byte) error {
	eventID := uuid.NewString()
	query := `INSERT INTO outbox_events (id, topic, payload) VALUES ($1, $2, $3)`

	if _, err := r.storage.Pool.Exec(ctx, query, eventID, topic, payload); err != nil {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}
	return nil
}

// SaveInTx сохраняет appointment и событие в одной транзакции.
func (r *PostgresOutboxRepository) SaveInTx(
	ctx context.Context,
	a *models.Appointment,
	topic string,
	payload []byte,
) error {
	log.Printf("[SaveInTx] START id=%s topic=%s", a.ID, topic)

	// Начать транзакцию
	tx, err := r.storage.Pool.Begin(ctx)
	if err != nil {
		log.Printf("[SaveInTx] BEGIN failed: %v", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Откатить при ошибке (если Commit уже прошёл — Rollback вернёт ErrTxClosed, это не страшно)
	defer tx.Rollback(ctx)

	// 1. Вставить appointment
	insertApptQuery := `INSERT INTO appointments (id, patient_id, doctor, time, status)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING created_at`

	if err := tx.QueryRow(
		ctx, insertApptQuery,
		a.ID, a.PatientID, a.Doctor, a.Time, a.Status,
	).Scan(&a.CreatedAt); err != nil {
		log.Printf("[SaveInTx] INSERT appointment failed: %v", err)
		return fmt.Errorf("failed to insert appointment: %w", err)
	}
	log.Printf("[SaveInTx] appointment inserted id=%s", a.ID)

	// 2. Вставить событие в outbox
	eventID := uuid.NewString()
	insertEventQuery := `INSERT INTO outbox_events (id, topic, payload)
	VALUES ($1, $2, $3)`

	if _, err := tx.Exec(ctx, insertEventQuery, eventID, topic, payload); err != nil {
		log.Printf("[SaveInTx] INSERT outbox failed: %v", err)
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}
	log.Printf("[SaveInTx] outbox event inserted id=%s topic=%s", eventID, topic)

	// 3. Зафиксировать транзакцию
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[SaveInTx] COMMIT failed: %v", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	log.Printf("[SaveInTx] COMMIT OK id=%s", a.ID)
	return nil
}

// GetUnsent возвращает неотправленные события.
func (r *PostgresOutboxRepository) GetUnsent(ctx context.Context, limit int) ([]models.OutboxEvent, error) {
	query := `SELECT id, topic, payload, created_at
	FROM outbox_events
	WHERE sent_at IS NULL
	ORDER BY created_at ASC
	LIMIT $1`

	rows, err := r.storage.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query unsent events: %w", err)
	}
	defer rows.Close()

	var events []models.OutboxEvent
	for rows.Next() {
		var e models.OutboxEvent
		if err := rows.Scan(&e.ID, &e.Topic, &e.Payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan outbox event: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return events, nil
}

// MarkSent помечает событие как отправленное.
func (r *PostgresOutboxRepository) MarkSent(ctx context.Context, id string) error {
	query := `UPDATE outbox_events SET sent_at = $1 WHERE id = $2`
	now := time.Now().UTC()

	result, err := r.storage.Pool.Exec(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark event as sent: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("event %s not found", id)
	}
	return nil
}
