package config

import "os"

// Config хранит настройки patient-service.
type Config struct {
	GRPCPort    string
	DatabaseURL string
}

// Load читает настройки из переменных окружения.
// Если переменная не задана — использует значение по умолчанию.
func Load() Config {
	return Config{
		GRPCPort:    getEnv("GRPC_PORT", ":50051"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://patient_user:patient_pass@localhost:5432/patients_db?sslmode=disable"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
