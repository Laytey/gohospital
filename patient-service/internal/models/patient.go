package models

import "time"

// Patient — модель пациента из БД.
//
// ID — UUID в строковом формате (генерируется при создании).
type Patient struct {
	ID        string
	Name      string
	Birthdate time.Time
	CreatedAt time.Time
}
