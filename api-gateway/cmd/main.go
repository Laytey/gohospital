package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/Laytey/gohospital/api-gateway/docs"
	"github.com/Laytey/gohospital/api-gateway/internal/client"
	"github.com/Laytey/gohospital/api-gateway/internal/config"
	"github.com/Laytey/gohospital/api-gateway/internal/handlers"
	"github.com/Laytey/gohospital/api-gateway/internal/middleware"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const shutdownTimeout = 10 * time.Second

// @title           GoHospital API
// @version         1.0
// @description     Микросервисная система для автоматизации записей к врачу
// @host            localhost:8080
// @BasePath        /
func main() {
	// 1. Контекст, отменяемый при SIGTERM/SIGINT
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// 2. Конфиг
	cfg := config.Load()
	log.Printf("Config: HTTP Port=%s", cfg.HTTPPort)

	// 3. gRPC-клиенты
	patientClient, err := client.NewPatientClient(cfg.PatientServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create patient client: %v", err)
	}
	defer patientClient.Close()
	log.Println("Connected to patient-service")

	appointmentClient, err := client.NewAppointmentClient(cfg.AppointmentServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create appointment client: %v", err)
	}
	defer appointmentClient.Close()
	log.Println("Connected to appointment-service")

	// 4. Хендлеры
	patientHandler := handlers.NewPatientHandler(patientClient)
	appointmentHandler := handlers.NewAppointmentHandler(appointmentClient)

	// 5. Gin-роутер
	r := gin.New()
	r.Use(gin.Recovery()) //gin.Recovery() — это встроенная защита Gin
	r.Use(middleware.LoggingMiddleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/patients", patientHandler.CreatePatient)
	r.GET("/patients/:id", patientHandler.GetPatient)

	r.POST("/appointments", appointmentHandler.CreateAppointment)
	r.GET("/appointments/:id", appointmentHandler.GetAppointment)
	r.POST("/appointments/:id/cancel", appointmentHandler.CancelAppointment)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 6. http.Server вместо r.Run()
	srv := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: r,
	}

	// 7. Запуск в горутине
	go func() {
		log.Printf("HTTP server listening on %s", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// 8. Ждём сигнал
	<-ctx.Done()
	log.Println("Shutdown signal received, starting graceful shutdown...")

	// 9. Shutdown с таймаутом
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server forced to shutdown: %v", err)
	} else {
		log.Println("HTTP server stopped gracefully")
	}

	log.Println("Goodbye")
}
