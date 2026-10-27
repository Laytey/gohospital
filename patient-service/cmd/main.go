package main

import (
	"context"
	"log"
	"net"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/Laytey/gohospital/patient-service/internal/config"
	"github.com/Laytey/gohospital/patient-service/internal/db"
	"github.com/Laytey/gohospital/patient-service/internal/grpcserver"
	"github.com/Laytey/gohospital/patient-service/internal/repository"
	"github.com/Laytey/gohospital/patient-service/migrations"
	patientpb "github.com/Laytey/gohospital/proto/patient"
)

const shutdownTimeout = 10 * time.Second

func main() {
	// 1. Контекст, отменяемый при SIGTERM/SIGINT
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	//SIGINT (Signal Interrupt) - Ctrl+C в терминале
	//SIGTERM (Signal Terminate) - сигнал, который посылает Docker (команда docker compose stop),
	//когда хочет попросить контейнер выключиться.

	// 2. Конфиг
	cfg := config.Load()
	log.Printf("Config: GRPCPort=%s", cfg.GRPCPort)

	// 3. Подключение к PostgreSQL (с retry)
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

	// 5. Собрать зависимости
	patientRepo := repository.NewPostgresPatientRepository(storage)
	server := grpcserver.NewPatientServer(patientRepo)

	// 6. gRPC-сервер
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcserver.RecoveryInterceptor,
			grpcserver.LoggingInterceptor,
		),
	)
	patientpb.RegisterPatientServiceServer(grpcServer, server)
	// передает запросы по методам, ктоорые сгенрировал протобаф

	listener, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", cfg.GRPCPort, err)
	}

	// 7. Запустить Serve в горутине
	go func() {
		log.Printf("gRPC server listening on %s", cfg.GRPCPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC serve stopped: %v", err)
		}
	}()

	// 8. Ждём сигнал
	<-ctx.Done()
	log.Println("Shutdown signal received, starting graceful shutdown...")

	// 9. GracefulStop с таймаутом
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

	log.Println("Goodbye")
}
