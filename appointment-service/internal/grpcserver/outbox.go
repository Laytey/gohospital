package grpcserver

import (
	"context"
	"log"
	"time"

	"github.com/Laytey/gohospital/appointment-service/internal/repository"
	"github.com/nats-io/nats.go"
)

const (
	outboxBatchSize    = 100
	outboxPollInterval = 2 * time.Second
)

// StartOutboxPublisher запускает фоновый воркер, который читает неотправленные
// события из outbox и публикует их в NATS.
//
// Работает в отдельной горутине. Останавливается при отмене ctx.
func StartOutboxPublisher(ctx context.Context, repo repository.OutboxRepository, nc *nats.Conn) {
	go func() {
		log.Println("[OUTBOX] Publisher запущен")
		ticker := time.NewTicker(outboxPollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("[OUTBOX] Publisher остановлен")
				return
			case <-ticker.C:
				publishBatch(ctx, repo, nc)
			}
		}
	}()
}

// publishBatch забирает пачку неотправленных событий и публикует их.
func publishBatch(ctx context.Context, repo repository.OutboxRepository, nc *nats.Conn) {
	events, err := repo.GetUnsent(ctx, outboxBatchSize)
	if err != nil {
		log.Printf("[OUTBOX] failed to fetch unsent events: %v", err)
		return
	}

	for _, e := range events {
		if err := nc.Publish(e.Topic, e.Payload); err != nil {
			log.Printf("[OUTBOX] failed to publish event %s: %v", e.ID, err)
			// Не помечаем как отправленное — попробуем в следующий раз
			continue
		}
		// Убедимся, что сообщение дошло до сервера, а не только в буфер
		if err := nc.Flush(); err != nil {
			log.Printf("[OUTBOX] flush failed: %v", err)
			continue
		}
		if err := repo.MarkSent(ctx, e.ID); err != nil {
			log.Printf("[OUTBOX] failed to mark event %s as sent: %v", e.ID, err)
		} else {
			log.Printf("[OUTBOX] event %s published to %s", e.ID, e.Topic)
		}
	}
}
