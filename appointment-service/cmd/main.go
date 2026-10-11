package main

import (
	"context"
	"log"
	"net"

	"github.com/Laytey/gohospital/appointment-service/internal/config"
	"github.com/Laytey/gohospital/appointment-service/internal/db"
	"github.com/Laytey/gohospital/appointment-service/internal/grpcserver"
	"github.com/Laytey/gohospital/appointment-service/internal/repository"
	appointmentpb "github.com/Laytey/gohospital/proto/appointment"
	patientpb "github.com/Laytey/gohospital/proto/patient"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 1. Загрузить конфиг
	cfg := config.Load()
	log.Printf("Config: GRPCPort=%s, NatsURL=%s", cfg.GRPCPort, cfg.NatsURL)

	// 2. Подключиться к PostgreSQL (appointments_db)
	storage, err := db.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	// 3. Создать репозиторий
	appointmentRepo := repository.NewPostgresAppointmentRepository(storage)

	// 4. Подключиться к NATS
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.Fatalf("NATS connection failed: %v", err)
	}
	defer nc.Close()
	log.Println("Connected to NATS")

	// 5. Запустить notification worker
	if err := grpcserver.StartNotificationWorker(nc); err != nil {
		log.Fatalf("Failed to start notification worker: %v", err)
	}

	// 6. Подключиться к patient-service (gRPC-клиент)
	// Адрес patient-service: localhost:50051
	// Insecure credentials — без TLS (для локальной разработки)
	patientConn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to patient-service: %v", err)
	}
	defer patientConn.Close()

	patientClient := patientpb.NewPatientServiceClient(patientConn)
	log.Println("Connected to patient-service (gRPC client)")

	// 7. Создать gRPC-сервер с зависимостями
	server := grpcserver.NewAppointmentServer(appointmentRepo, nc, patientClient)

	// 8. Создать TCP-листенер
	listener, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", cfg.GRPCPort, err)
	}

	// 9. Создать gRPC-сервер (стандартный)
	grpcServer := grpc.NewServer()
	appointmentpb.RegisterAppointmentServiceServer(grpcServer, server)

	log.Printf("gRPC server listening on %s", cfg.GRPCPort)

	// 10. Запустить
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Server failed to serve: %v", err)
	}
}
