package models

import "time"

// OutboxEvent — событие, ожидающее публикации в NATS.
//
// Записывается в БД в одной транзакции с бизнес-данными (например, с записью
// на приём). Фоновый воркер читает неотправленные события и публикует их.
type OutboxEvent struct {
	ID        string
	Topic     string
	Payload   []byte
	CreatedAt time.Time
	SentAt    *time.Time
}
