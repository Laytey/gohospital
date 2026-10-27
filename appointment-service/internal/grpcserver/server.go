package grpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/Laytey/gohospital/appointment-service/internal/models"
	"github.com/Laytey/gohospital/appointment-service/internal/repository"
	appointmentpb "github.com/Laytey/gohospital/proto/appointment"
	patientpb "github.com/Laytey/gohospital/proto/patient"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	statusCreated  = "created"
	statusCanceled = "canceled"
)

// AppointmentServer реализует gRPC-сервис AppointmentService.
//
// Хранит зависимости:
//   - repo — работа с БД appointments;
//   - nc — NATS для публикации событий appointment.created/canceled;
//   - patientClient — gRPC-клиент к patient-service (проверка пациента).
type AppointmentServer struct {
	appointmentpb.UnimplementedAppointmentServiceServer
	repo          repository.AppointmentRepository
	outboxRepo    repository.OutboxRepository
	nc            *nats.Conn
	patientClient patientpb.PatientServiceClient
}

// NewAppointmentServer создаёт gRPC-сервер с указанными зависимостями.
func NewAppointmentServer(
	repo repository.AppointmentRepository,
	outboxRepo repository.OutboxRepository,
	nc *nats.Conn,
	patientClient patientpb.PatientServiceClient,
) *AppointmentServer {
	return &AppointmentServer{
		repo:          repo,
		outboxRepo:    outboxRepo,
		nc:            nc,
		patientClient: patientClient,
	}
}

// CreateAppointment обрабатывает gRPC-вызов создания записи.
func (s *AppointmentServer) CreateAppointment(ctx context.Context, req *appointmentpb.CreateAppointmentRequest) (*appointmentpb.Appointment, error) {
	// 1. Валидация входных данных
	patientID := strings.TrimSpace(req.GetPatientId())
	if patientID == "" {
		return nil, status.Error(codes.InvalidArgument, "ID пациента не может быть пустым")
	}

	doctor := strings.TrimSpace(req.GetDoctor())
	if doctor == "" {
		return nil, status.Error(codes.InvalidArgument, "имя врача не может быть пустым")
	}

	appointmentTime, err := time.Parse(time.RFC3339, req.GetTime())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "неверный формат времени, ожидается RFC3339: %v", err)
	}
	if appointmentTime.Before(time.Now()) {
		return nil, status.Error(codes.InvalidArgument, "время приёма не может быть в прошлом")
	}

	// 2. Проверить существование пациента через gRPC
	// Отдельный deadline для вызова patient-service:
	// меньше, чем у api-gateway, чтобы у appointment было время
	// сохранить запись в БД после ответа пациента.
	patientCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err = s.patientClient.GetPatient(patientCtx, &patientpb.GetPatientRequest{Id: patientID})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				return nil, status.Errorf(codes.NotFound, "пациент %s не найден", patientID)
			case codes.InvalidArgument:
				return nil, status.Errorf(codes.InvalidArgument, "некорректный ID пациента: %s", st.Message())
			case codes.DeadlineExceeded:
				// Наш собственный таймаут на вызов patient-service истёк.
				// Прокидываем наверх, чтобы api-gateway вернул 504, а не 500.
				return nil, status.Error(codes.DeadlineExceeded, "превышено время ожидания ответа от patient-service")
			case codes.Unavailable:
				// patient-service недоступен (контейнер лёг, сеть порвалась).
				return nil, status.Error(codes.Unavailable, "patient-service временно недоступен")
			}
		}
		return nil, status.Errorf(codes.Internal, "не удалось проверить пациента: %v", err)
	}

	// 3. Создать и сохранить
	appointment := models.Appointment{
		ID:        uuid.NewString(),
		PatientID: patientID,
		Doctor:    doctor,
		Time:      appointmentTime,
		Status:    statusCreated,
	}

	// 4. Сериализовать событие в JSON
	payload, err := json.Marshal(map[string]any{
		"id":         appointment.ID,
		"patient_id": appointment.PatientID,
		"doctor":     appointment.Doctor,
		"time":       appointment.Time.Format(time.RFC3339),
		"status":     appointment.Status,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "не удалось сериализовать событие: %v", err)
	}

	// 5. Сохранить appointment + событие в одной транзакции
	if err := s.outboxRepo.SaveInTx(ctx, &appointment, "appointment.created", payload); err != nil {
		return nil, status.Errorf(codes.Internal, "не удалось сохранить запись: %v", err)
	}

	return toProtoAppointment(appointment), nil
}

// GetAppointment обрабатывает gRPC-вызов получения записи по ID.
func (s *AppointmentServer) GetAppointment(ctx context.Context, req *appointmentpb.GetAppointmentRequest) (*appointmentpb.Appointment, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "ID не может быть пустым")
	}

	a, err := s.repo.GetByID(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, repository.ErrAppointmentNotFound) {
			return nil, status.Errorf(codes.NotFound, "запись %s не найдена", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "не удалось получить запись: %v", err)
	}

	return toProtoAppointment(a), nil
}

// CancelAppointment обрабатывает gRPC-вызов отмены записи.
func (s *AppointmentServer) CancelAppointment(ctx context.Context, req *appointmentpb.CancelAppointmentRequest) (*appointmentpb.Appointment, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "ID не может быть пустым")
	}

	// 1. Обновить статус
	err := s.repo.UpdateStatus(ctx, req.GetId(), statusCanceled)
	if err != nil {
		if errors.Is(err, repository.ErrAppointmentNotFound) {
			return nil, status.Errorf(codes.NotFound, "запись %s не найдена", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "не удалось обновить статус записи: %v", err)
	}

	// 2. Прочитать обновлённую запись
	a, err := s.repo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "не удалось прочитать обновлённую запись: %v", err)
	}

	// 3. Опубликовать событие через outbox
	payload, err := json.Marshal(map[string]any{
		"id":         a.ID,
		"patient_id": a.PatientID,
		"doctor":     a.Doctor,
		"time":       a.Time.Format(time.RFC3339),
		"status":     a.Status,
	})
	if err != nil {
		log.Printf("failed to marshal canceled event: %v", err)
		return toProtoAppointment(a), nil // возвращаем результат, но без события
	}

	if err := s.outboxRepo.SaveEventOnly(ctx, "appointment.canceled", payload); err != nil {
		log.Printf("failed to save canceled event to outbox: %v", err)
	}

	return toProtoAppointment(a), nil

}

// toProtoAppointment преобразует модель БД в protobuf-структуру.
func toProtoAppointment(a models.Appointment) *appointmentpb.Appointment {
	return &appointmentpb.Appointment{
		Id:        a.ID,
		PatientId: a.PatientID,
		Doctor:    a.Doctor,
		Time:      a.Time.Format(time.RFC3339),
		Status:    a.Status,
	}
}
