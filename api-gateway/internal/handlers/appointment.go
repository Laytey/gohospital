package handlers

import (
	"net/http"

	"github.com/Laytey/gohospital/api-gateway/internal/client"
	_ "github.com/Laytey/gohospital/proto/appointment"
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
// Тело запроса — JSON с полями name, birthdate.
//
// Ответы:
//   - 400 Bad Request — если JSON некорректен.
//   - 500 Internal Server Error — при внутренней ошибке.
//   - 201 Created — созданная запись.
//
// @Summary      Создать запись
// @Description  Добавляет новую запись в базу данных
// @Tags         appointments
// @Accept       json
// @Produce      json
// @Param        patient  body      CreateAppointmentRequest  true  "Данные записи"
// @Success      201      {object}  appointmentpb.Appointment
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /appointments [post]
func (h *AppointmentHandler) CreateAppointment(c *gin.Context) {

	var req CreateAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "некорректное тело запроса: поля 'patient_id', 'doctor' и 'time' обязательны",
		})
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
// @Summary      Получить запись
// @Description  Возвращает запись по её ID
// @Tags         appointments
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "ID записи"
// @Success      200  {object}  appointmentpb.Appointment
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /appointments/{id} [get]
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
// @Summary      Отменить запись
// @Description  Меняет статус записи на 'canceled'
// @Tags         appointments
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "ID записи"
// @Success      200  {object}  appointmentpb.Appointment
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /appointments/{id}/cancel [post]
func (h *AppointmentHandler) CancelAppointment(c *gin.Context) {
	id := c.Param("id")
	appointment, err := h.client.CancelAppointment(c.Request.Context(), id)
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, appointment)
}
