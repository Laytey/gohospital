package main

import (
	"context"
	"log"
	"net"
	"os/signal"
	"syscall"
	"time"

	"github.com/Laytey/gohospital/appointment-service/internal/config"
	"github.com/Laytey/gohospital/appointment-service/internal/db"
	"github.com/Laytey/gohospital/appointment-service/internal/grpcserver"
	"github.com/Laytey/gohospital/appointment-service/internal/repository"
	"github.com/Laytey/gohospital/appointment-service/migrations"
	appointmentpb "github.com/Laytey/gohospital/proto/appointment"
	patientpb "github.com/Laytey/gohospital/proto/patient"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const shutdownTimeout = 10 * time.Second

func main() {
	// 1. Контекст, отменяемый при SIGTERM/SIGINT
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// 2. Конфиг
	cfg := config.Load()
	log.Printf("Config: GRPCPort=%s, NatsURL=%s, PatientServiceAddr=%s",
		cfg.GRPCPort, cfg.NatsURL, cfg.PatientServiceAddr)

	// 3. PostgreSQL
	storage, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer storage.Pool.Close()
	log.Println("Connected to PostgreSQL")

	// 4. Миграции
	if err := db.RunMigrations(ctx, cfg.DatabaseURL, migrations.FS); err != nil {
		log.Fatalf("Migrations failed: %v", err)
	}

	// 5. NATS
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.Fatalf("NATS connection failed: %v", err)
	}
	defer nc.Close()
	log.Println("Connected to NATS")

	// 6. Notification worker
	if err := grpcserver.StartNotificationWorker(nc); err != nil {
		log.Fatalf("Failed to start notification worker: %v", err)
	}

	// 7. gRPC-клиент к patient-service
	log.Printf("Connecting to patient-service at %s", cfg.PatientServiceAddr)
	patientConn, err := grpc.NewClient(
		cfg.PatientServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to patient-service: %v", err)
	}
	defer patientConn.Close()
	patientClient := patientpb.NewPatientServiceClient(patientConn)
	log.Println("Connected to patient-service (gRPC client)")

	// 8. Наш gRPC-сервер
	appointmentRepo := repository.NewPostgresAppointmentRepository(storage)
	outboxRepo := repository.NewPostgresOutboxRepository(storage)

	// Запустить воркер Outbox
	grpcserver.StartOutboxPublisher(ctx, outboxRepo, nc)

	server := grpcserver.NewAppointmentServer(appointmentRepo, outboxRepo, nc, patientClient)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcserver.RecoveryInterceptor,
			grpcserver.LoggingInterceptor,
		),
	)
	appointmentpb.RegisterAppointmentServiceServer(grpcServer, server)

	listener, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", cfg.GRPCPort, err)
	}

	go func() {
		log.Printf("gRPC server listening on %s", cfg.GRPCPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC serve stopped: %v", err)
		}
	}()

	// 9. Ждём сигнал
	<-ctx.Done()
	log.Println("Shutdown signal received, starting graceful shutdown...")

	// 10. Сначала gRPC — чтобы не принимали новые запросы
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("gRPC server stopped gracefully")
	case <-time.After(shutdownTimeout):
		log.Println("Shutdown timeout exceeded, forcing stop")
		grpcServer.Stop()
	}

	// 11. Дренаж NATS — дошли буферизованные сообщения
	if err := nc.Drain(); err != nil {
		log.Printf("NATS drain error: %v", err)
	} else {
		log.Println("NATS drained")
	}

	log.Println("Goodbye")
}
