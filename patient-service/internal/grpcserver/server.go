package grpcserver

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Laytey/gohospital/patient-service/internal/models"
	"github.com/Laytey/gohospital/patient-service/internal/repository"
	patientpb "github.com/Laytey/gohospital/proto/patient"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	dateLayout = "2006-01-02"
	maxAge     = 150
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
	// 1. Валидация
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "имя не может быть пустым")
	}

	birthdate, err := time.Parse(dateLayout, req.GetBirthdate())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "неверный формат даты рождения, ожидается ГГГГ-ММ-ДД: %v", err)
	}

	now := time.Now().UTC()
	if birthdate.After(now) {
		return nil, status.Error(codes.InvalidArgument, "дата рождения не может быть в будущем")
	}

	age := now.Year() - birthdate.Year()
	if age > maxAge {
		return nil, status.Errorf(codes.InvalidArgument, "возраст %d превышает максимально допустимый (%d лет)", age, maxAge)
	}

	// 2. Сохранить
	patient := models.Patient{Name: name, Birthdate: birthdate}
	if err := s.repo.Save(ctx, &patient); err != nil {
		return nil, status.Errorf(codes.Internal, "не удалось сохранить пациента: %v", err)
	}

	return toProtoPatient(patient), nil
}

// GetPatient обрабатывает gRPC-вызов получения пациента по ID.
func (s *PatientServer) GetPatient(ctx context.Context, req *patientpb.GetPatientRequest) (*patientpb.Patient, error) {
	// 1. Валидация ID (пустой UUID — точно ошибка клиента)
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "ID не может быть пустым")
	}

	if _, err := uuid.Parse(req.GetId()); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "некорректный формат UUID: %s", req.GetId())
	}

	// 2. Получить из БД
	patient, err := s.repo.GetByID(ctx, req.GetId())
	if err != nil {
		// 2.1. Реально нет записи - NotFound
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "пациент %s не найден", req.GetId())
		}
		// 2.2. Всё остальное (проблема с БД, сетью, контекстом) - Internal
		return nil, status.Errorf(codes.Internal, "не удалось получить пациента: %v", err)
	}

	return toProtoPatient(patient), nil
}

// toProtoPatient преобразует модель БД в protobuf-структуру.
func toProtoPatient(p models.Patient) *patientpb.Patient {
	return &patientpb.Patient{
		Id:        p.ID,
		Name:      p.Name,
		Birthdate: p.Birthdate.Format(dateLayout),
	}
}
