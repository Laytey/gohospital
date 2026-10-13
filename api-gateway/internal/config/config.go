package config

import "os"

// Config хранит настройки api-gateway.
type Config struct {
	HTTPPort               string
	PatientServiceAddr     string
	AppointmentServiceAddr string
}

// Load читает настройки из переменных окружения.
func Load() Config {
	return Config{
		HTTPPort:               getEnv("HTTP_PORT", ":8080"),
		PatientServiceAddr:     getEnv("PATIENT_SERVICE_ADDR", "localhost:50051"),
		AppointmentServiceAddr: getEnv("APPOINTMENT_SERVICE_ADDR", "localhost:50052"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
