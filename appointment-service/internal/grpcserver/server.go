package grpcserver

import (
	"context"
	"log"
	"time"

	"github.com/Laytey/gohospital/appointment-service/internal/models"
	"github.com/Laytey/gohospital/appointment-service/internal/repository"
	appointmentpb "github.com/Laytey/gohospital/proto/appointment"
	patientpb "github.com/Laytey/gohospital/proto/patient"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	nc            *nats.Conn
	patientClient patientpb.PatientServiceClient
}

// NewAppointmentServer создаёт gRPC-сервер с указанными зависимостями.
func NewAppointmentServer(
	repo repository.AppointmentRepository,
	nc *nats.Conn,
	patientClient patientpb.PatientServiceClient,
) *AppointmentServer {
	return &AppointmentServer{
		repo:          repo,
		nc:            nc,
		patientClient: patientClient,
	}
}

// CreateAppointment обрабатывает gRPC-вызов создания записи.
func (s *AppointmentServer) CreateAppointment(ctx context.Context, req *appointmentpb.CreateAppointmentRequest) (*appointmentpb.Appointment, error) {
	// 1. Проверить существование пациента через gRPC
	_, err := s.patientClient.GetPatient(ctx, &patientpb.GetPatientRequest{Id: req.GetPatientId()})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "patient not found: %v", err)
	}

	// 2. Распарсить время
	appointmentTime, err := time.Parse(time.RFC3339, req.GetTime())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid time format: %v", err)
	}

	// 3. Создать модель
	appointment := models.Appointment{
		PatientID: req.GetPatientId(),
		Doctor:    req.GetDoctor(),
		Time:      appointmentTime,
		Status:    "created",
	}

	// 4. Сохранить в БД
	if err := s.repo.Save(ctx, &appointment); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save appointment: %v", err)
	}

	// 5. Опубликовать событие в NATS
	if err := s.nc.Publish("appointment.created", []byte(appointment.ID)); err != nil {
		log.Printf("failed to publish appointment.created: %v", err)
	}

	// 6. Вернуть
	return toProtoAppointment(appointment), nil
}

// GetAppointment обрабатывает gRPC-вызов получения записи по ID.
func (s *AppointmentServer) GetAppointment(ctx context.Context, req *appointmentpb.GetAppointmentRequest) (*appointmentpb.Appointment, error) {

	a, err := s.repo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "appointment not found: %v", err)
	}

	return toProtoAppointment(a), nil
}

// CancelAppointment обрабатывает gRPC-вызов отмены записи.
func (s *AppointmentServer) CancelAppointment(ctx context.Context, req *appointmentpb.CancelAppointmentRequest) (*appointmentpb.Appointment, error) {
	// 1. Обновить статус в БД
	err := s.repo.UpdateStatus(ctx, req.GetId(), "canceled")
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "appointment not found: %v", err)
	}

	// 2. Прочитать обновлённую запись
	a, err := s.repo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get appointment: %v", err)
	}

	// 3. Опубликовать событие в NATS
	err = s.nc.Publish("appointment.canceled", []byte(a.ID))
	if err != nil {
		log.Printf("failed to publish: %v", err)
	}

	// 4. Вернуть прото
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
