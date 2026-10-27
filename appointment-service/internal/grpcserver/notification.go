package grpcserver

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

// NotificationEvent — payload, который приходит из NATS.
type NotificationEvent struct {
	ID        string `json:"id"`
	PatientID string `json:"patient_id"`
	Doctor    string `json:"doctor"`
	Time      string `json:"time"`
	Status    string `json:"status"`
}

// StartNotificationWorker запускает фоновый обработчик уведомлений.
//
// Подписывается на события appointment.created и appointment.canceled,
// логирует каждое событие. Это заглушка — в реальном проекте здесь
// была бы отправка email/SMS/пуш-уведомлений.
func StartNotificationWorker(nc *nats.Conn) error {
	_, err := nc.Subscribe("appointment.created", func(msg *nats.Msg) {
		var e NotificationEvent
		if err := json.Unmarshal(msg.Data, &e); err != nil {
			log.Printf("[NOTIFICATION] failed to parse appointment.created: %v", err)
			return
		}
		log.Printf("[NOTIFICATION] Запись создана: id=%s, patient=%s, doctor=%s, time=%s",
			e.ID, e.PatientID, e.Doctor, e.Time)
	})
	if err != nil {
		return err
	}

	_, err = nc.Subscribe("appointment.canceled", func(msg *nats.Msg) {
		var e NotificationEvent
		if err := json.Unmarshal(msg.Data, &e); err != nil {
			log.Printf("[NOTIFICATION] failed to parse appointment.canceled: %v", err)
			return
		}
		log.Printf("[NOTIFICATION] CANCELED-топик: id=%s, patient=%s", e.ID, e.PatientID)
	})
	if err != nil {
		return err
	}

	log.Println("[NOTIFICATION] Worker запущен, слушаем события...")
	return nil
}
