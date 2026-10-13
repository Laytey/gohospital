package handlers

import (
	"net/http"

	"github.com/Laytey/gohospital/api-gateway/internal/client"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PatientHandler обрабатывает HTTP-запросы для пациентов.
//
// Делегирует все операции в PatientClient (gRPC-клиент к patient-service).
type PatientHandler struct {
	client *client.PatientClient
}

// NewPatientHandler создаёт хендлер с указанным gRPC-клиентом.
func NewPatientHandler(c *client.PatientClient) *PatientHandler {
	return &PatientHandler{
		client: c,
	}
}

// CreatePatientRequest — тело HTTP-запроса на создание пациента.
type CreatePatientRequest struct {
	Name      string `json:"name" binding:"required"`
	Birthdate string `json:"birthdate" binding:"required"`
}

// CreatePatient обрабатывает POST /patients.
//
// Тело запроса — JSON с полями name, birthdate.
//
// Ответы:
//   - 400 Bad Request — если JSON некорректен.
//   - 500 Internal Server Error — при внутренней ошибке.
//   - 201 Created — созданный пациент.
func (h *PatientHandler) CreatePatient(c *gin.Context) {
	var req CreatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	patient, err := h.client.CreatePatient(c.Request.Context(), req.Name, req.Birthdate)
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, patient)
}

// GetPatient обрабатывает GET /patients/:id.
//
// Ответы:
//   - 404 Not Found — если пациент не найден.
//   - 500 Internal Server Error — при внутренней ошибке.
//   - 200 OK — пациент в JSON.
func (h *PatientHandler) GetPatient(c *gin.Context) {
	id := c.Param("id")

	patient, err := h.client.GetPatient(c.Request.Context(), id)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, patient)
}

// handleGRPCError преобразует gRPC-ошибку в HTTP-ответ.
//
// Маппинг кодов gRPC → HTTP:
//   - NotFound         → 404
//   - InvalidArgument  → 400
//   - Internal, Unknown → 500
func handleGRPCError(c *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	switch st.Code() {
	case codes.NotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": st.Message()})
	case codes.InvalidArgument:
		c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": st.Message()})
	}
}
