package main

import (
	"log"

	"github.com/Laytey/gohospital/api-gateway/internal/client"
	"github.com/Laytey/gohospital/api-gateway/internal/config"
	"github.com/Laytey/gohospital/api-gateway/internal/handlers"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Конфиг
	cfg := config.Load()
	log.Printf("Config: HTTP Port=%s", cfg.HTTPPort)

	// 2. gRPC-клиент к patient-service
	patientClient, err := client.NewPatientClient(cfg.PatientServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create patient client: %v", err)
	}
	defer patientClient.Close()
	log.Printf("Connected to patient-service")

	// 3. gRPC-клиент к appointment-service
	appointmentClient, err := client.NewAppointmentClient(cfg.AppointmentServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create appointment client: %v", err)
	}
	defer appointmentClient.Close()
	log.Printf("Connected to appointment-service")

	// 4. Создать хендлеры
	patientHandler := handlers.NewPatientHandler(patientClient)
	appointmentHandler := handlers.NewAppointmentHandler(appointmentClient)

	// 5. Gin-роутер
	r := gin.Default()

	// Health-check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Пациенты
	r.POST("/patients", patientHandler.CreatePatient)
	r.GET("/patients/:id", patientHandler.GetPatient)

	// Записи
	r.POST("/appointments", appointmentHandler.CreateAppointment)
	r.GET("/appointments/:id", appointmentHandler.GetAppointment)
	r.POST("/appointments/:id/cancel", appointmentHandler.CancelAppointment)

	// 6. Запуск HTTP-сервера
	log.Printf("HTTP server listening on %s", cfg.HTTPPort)
	if err := r.Run(cfg.HTTPPort); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
