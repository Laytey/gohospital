package client

import (
	"context"
	"fmt"
	"time"

	patientpb "github.com/Laytey/gohospital/proto/patient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// PatientClient — обёртка над gRPC-клиентом patient-service.
//
// Инкапсулирует соединение и предоставляет удобные методы
// для api-gateway.
type PatientClient struct {
	conn    *grpc.ClientConn
	client  patientpb.PatientServiceClient
	timeout time.Duration
}

// NewPatientClient создаёт клиент и устанавливает соединение с patient-service.
//
// addr — адрес сервиса, например "localhost:50051".
func NewPatientClient(addr string) (*PatientClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to patient-service: %v", err)
	}

	return &PatientClient{
		conn:    conn,
		client:  patientpb.NewPatientServiceClient(conn),
		timeout: defaultTimeout,
	}, nil

}

// Close закрывает соединение с patient-service.
func (c *PatientClient) Close() error {
	return c.conn.Close()
}

// CreatePatient создаёт нового пациента.
func (c *PatientClient) CreatePatient(ctx context.Context, name, birthdate string) (*patientpb.Patient, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.CreatePatient(ctx, &patientpb.CreatePatientRequest{
		Name:      name,
		Birthdate: birthdate,
	})
}

// GetPatient возвращает пациента по ID.
func (c *PatientClient) GetPatient(ctx context.Context, id string) (*patientpb.Patient, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.GetPatient(ctx, &patientpb.GetPatientRequest{
		Id: id,
	})
}
