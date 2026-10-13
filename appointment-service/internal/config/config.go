package config

import "os"

// Config хранит настройки appointment-service.
type Config struct {
	GRPCPort           string
	DatabaseURL        string
	NatsURL            string
	PatientServiceAddr string
}

// Load читает настройки из переменных окружения.
func Load() Config {
	return Config{
		GRPCPort:           getEnv("GRPC_PORT", ":50052"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://appointment_user:appointment_pass@localhost:5433/appointments_db?sslmode=disable"),
		NatsURL:            getEnv("NATS_URL", "nats://localhost:4222"),
		PatientServiceAddr: getEnv("PATIENT_SERVICE_ADDR", "localhost:50051"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
