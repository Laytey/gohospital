package grpcserver

import (
	"log"

	"github.com/nats-io/nats.go"
)

// StartNotificationWorker запускает фоновый обработчик уведомлений.
//
// Подписывается на события appointment.created и appointment.canceled,
// логирует каждое событие. Это заглушка — в реальном проекте здесь
// была бы отправка email/SMS/пуш-уведомлений.
func StartNotificationWorker(nc *nats.Conn) error {
	// Подписка на appointment.created
	_, err := nc.Subscribe("appointment.created", func(msg *nats.Msg) {
		log.Printf("[NOTIFICATION] Запись создана: appointment_id=%s", string(msg.Data))
	})
	if err != nil {
		return err
	}

	// Подписка на appointment.canceled
	_, err = nc.Subscribe("appointment.canceled", func(msg *nats.Msg) {
		log.Printf("[NOTIFICATION] Запись отменена: appointment_id=%s", string(msg.Data))
	})
	if err != nil {
		return err
	}

	log.Println("[NOTIFICATION] Worker запущен, слушаем события...")
	return nil
}
