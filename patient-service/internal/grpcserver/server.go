package grpc

import (
	"context"
	"time"

	"github.com/Laytey/gohospital/patient-service/internal/models"
	"github.com/Laytey/gohospital/patient-service/internal/repository"
	patientpb "github.com/Laytey/gohospital/proto/patient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PatientServer реализует gRPC-сервис PatientService.
type PatientServer struct {
	patientpb.UnimplementedPatientServiceServer
	repo repository.PatientRepository
}

// NewPatientServer создаёт gRPC-сервер с указанным репозиторием.
func NewPatientServer(repo repository.PatientRepository) *PatientServer {
	return &PatientServer{
		repo: repo,
	}
}

// CreatePatient обрабатывает gRPC-вызов создания пациента.
func (s *PatientServer) CreatePatient(ctx context.Context, req *patientpb.CreatePatientRequest) (*patientpb.Patient, error) {
	// 1. Распарсить дату
	birthdate, err := time.Parse("2006-01-02", req.GetBirthdate())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid birthdate format: %v", err)
	}

	// 2. Создать модель
	patient := models.Patient{
		Name:      req.GetName(),
		Birthdate: birthdate,
	}

	// 3. Сохранить в БД
	if err := s.repo.Save(ctx, &patient); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save patient: %v", err)
	}

	// 4. Вернуть protobuf
	return toProtoPatient(patient), nil

}

// GetPatient обрабатывает gRPC-вызов получения пациента по ID.
func (s *PatientServer) GetPatient(ctx context.Context, req *patientpb.GetPatientRequest) (*patientpb.Patient, error) {
	// 1. Получить из БД
	patient, err := s.repo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "patient not found: %v", err)
	}

	// 2. Вернуть protobuf
	return toProtoPatient(patient), nil
}

// toProtoPatient преобразует модель БД в protobuf-структуру.
func toProtoPatient(p models.Patient) *patientpb.Patient {
	return &patientpb.Patient{
		Id:        p.ID,
		Name:      p.Name,
		Birthdate: p.Birthdate.Format("2006-01-02"),
	}
}
