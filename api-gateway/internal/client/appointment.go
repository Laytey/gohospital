package client

import (
	"context"
	"fmt"

	appointmentpb "github.com/Laytey/gohospital/proto/appointment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AppointmentClient — обёртка над gRPC-клиентом appointment-service.
//
// Инкапсулирует соединение и предоставляет удобные методы
// для api-gateway.
type AppointmentClient struct {
	conn   *grpc.ClientConn
	client appointmentpb.AppointmentServiceClient
}

// NewAppointmentClient создаёт клиент и устанавливает соединение
// с appointment-service.
//
// addr — адрес сервиса, например "localhost:50052".
func NewAppointmentClient(addr string) (*AppointmentClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to appointment service: %v", err)
	}

	return &AppointmentClient{
		conn:   conn,
		client: appointmentpb.NewAppointmentServiceClient(conn),
	}, nil
}

// Close закрывает соединение с appointment-service.
func (c *AppointmentClient) Close() error {
	return c.conn.Close()
}

// CreateAppointment создаёт новую запись.
func (c *AppointmentClient) CreateAppointment(ctx context.Context, patientID, doctor, timeStr string) (*appointmentpb.Appointment, error) {
	return c.client.CreateAppointment(ctx, &appointmentpb.CreateAppointmentRequest{
		PatientId: patientID,
		Doctor:    doctor,
		Time:      timeStr,
	})
}

// GetAppointment возвращает запись по ID.
func (c *AppointmentClient) GetAppointment(ctx context.Context, id string) (*appointmentpb.Appointment, error) {
	return c.client.GetAppointment(ctx, &appointmentpb.GetAppointmentRequest{
		Id: id,
	})
}

// CancelAppointment отменяет запись.
func (c *AppointmentClient) CancelAppointment(ctx context.Context, id string) (*appointmentpb.Appointment, error) {
	return c.client.CancelAppointment(ctx, &appointmentpb.CancelAppointmentRequest{
		Id: id,
	})
}
