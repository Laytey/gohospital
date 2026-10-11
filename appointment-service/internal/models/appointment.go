package models

import "time"

// Appointment представляет запись пациента к врачу.
//
// Соответствует таблице appointments в БД.
// Поле Status принимает значения "created" или "canceled".

type Appointment struct {
	ID        string    // UUID
	PatientID string    // UUID пациента (ссылка на patient-service)
	Doctor    string    // ФИО врача
	Time      time.Time // Время приёма
	Status    string    // "created" или "canceled"
	CreatedAt time.Time // Дата создания записи
}
