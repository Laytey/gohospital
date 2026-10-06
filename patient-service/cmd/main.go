package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/Laytey/gohospital/patient-service/internal/config"
	"github.com/Laytey/gohospital/patient-service/internal/db"
	grpcserver "github.com/Laytey/gohospital/patient-service/internal/grpcserver"
	"github.com/Laytey/gohospital/patient-service/internal/repository"
	patientpb "github.com/Laytey/gohospital/proto/patient"
)

func main() {
	// 1. Загрузить конфиг
	cfg := config.Load()
	log.Printf("Config: GRPCPort=%s", cfg.GRPCPort)

	// 2. Подключиться к PostgreSQL
	storage, err := db.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	// 3. Создать репозиторий
	patientRepo := repository.NewPostgresPatientRepository(storage)

	// 4. Создать gRPC-сервер (наша реализация)
	server := grpcserver.NewPatientServer(patientRepo)

	// 5. Создать TCP-листенер
	listener, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", cfg.GRPCPort, err)
	}

	// 6. Создать gRPC-сервер (стандартный из библиотеки)
	grpcServer := grpc.NewServer()
	patientpb.RegisterPatientServiceServer(grpcServer, server)

	log.Printf("gRPC server listening on %s", cfg.GRPCPort)

	// 7. Запустить
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Serve failed: %v", err)
	}

}
