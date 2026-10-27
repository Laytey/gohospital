package handlers

import (
	"net/http"

	"github.com/Laytey/gohospital/api-gateway/internal/client"
	_ "github.com/Laytey/gohospital/proto/patient"
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
//
// @Summary      Создать пациента
// @Description  Добавляет нового пациента в базу данных
// @Tags         patients
// @Accept       json
// @Produce      json
// @Param        patient  body      CreatePatientRequest  true  "Данные пациента"
// @Success      201      {object}  patientpb.Patient
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /patients [post]
func (h *PatientHandler) CreatePatient(c *gin.Context) {
	var req CreatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "некорректное тело запроса: поля 'name' и 'birthdate' обязательны",
		})
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
// @Summary      Получить пациента
// @Description  Возвращает данные пациента по его ID
// @Tags         patients
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "ID пациента"
// @Success      200  {object}  patientpb.Patient
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /patients/{id} [get]
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
	st, ok := status.FromError(err) // 1. Пытаемся распаковать ошибку
	if !ok {                        // 2. Если ошибка НЕ от gRPC (например, ошибка сети внутри Go)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
		return
	}

	switch st.Code() { // 3. Смотрим на gRPC-код
	case codes.NotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": st.Message()})
	case codes.InvalidArgument:
		c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
	case codes.DeadlineExceeded:
		c.JSON(http.StatusGatewayTimeout, gin.H{
			"error": st.Message(),
		})
	case codes.Unavailable:
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": st.Message(),
		})
	default: // 4. Если код неизвестен (например, Internal)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
	}
}
