package handlers

import (
	"net/http"

	"github.com/Laytey/gohospital/api-gateway/internal/client"
	"github.com/gin-gonic/gin"
)

// AppointmentHandler обрабатывает HTTP-запросы для записей.
//
// Делегирует все операции в AppointmentClient
// (gRPC-клиент к appointment-service).
type AppointmentHandler struct {
	client *client.AppointmentClient
}

// NewAppointmentHandler создаёт хендлер с указанным gRPC-клиентом.
func NewAppointmentHandler(c *client.AppointmentClient) *AppointmentHandler {
	return &AppointmentHandler{
		client: c,
	}
}

// CreateAppointmentRequest — тело HTTP-запроса на создание записи.
type CreateAppointmentRequest struct {
	PatientID string `json:"patient_id" binding:"required"`
	Doctor    string `json:"doctor" binding:"required"`
	Time      string `json:"time" binding:"required"`
}

// CreateAppointment обрабатывает POST /appointments.
//
// Тело запроса — JSON с полями patient_id, doctor, time.
//
// Ответы:
//   - 400 Bad Request — если JSON некорректен или time невалиден.
//   - 404 Not Found — если пациент не найден.
//   - 500 Internal Server Error — при внутренней ошибке.
//   - 201 Created — созданная запись.
func (h *AppointmentHandler) CreateAppointment(c *gin.Context) {

	var req CreateAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	appointment, err := h.client.CreateAppointment(c.Request.Context(), req.PatientID, req.Doctor, req.Time)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, appointment)
}

// GetAppointment обрабатывает GET /appointments/:id.
//
// Ответы:
//   - 404 Not Found — если запись не найдена.
//   - 500 Internal Server Error — при внутренней ошибке.
//   - 200 OK — запись в JSON.
func (h *AppointmentHandler) GetAppointment(c *gin.Context) {
	id := c.Param("id")
	appointment, err := h.client.GetAppointment(c.Request.Context(), id)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, appointment)
}

// CancelAppointment обрабатывает POST /appointments/:id/cancel.
//
// Ответы:
//   - 404 Not Found — если запись не найдена.
//   - 500 Internal Server Error — при внутренней ошибке.
//   - 200 OK — обновлённая запись (status=canceled).
func (h *AppointmentHandler) CancelAppointment(c *gin.Context) {
	id := c.Param("id")
	appointment, err := h.client.CancelAppointment(c.Request.Context(), id)
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, appointment)
}
