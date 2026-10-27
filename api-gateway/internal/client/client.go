// Package client содержит gRPC-клиенты к внутренним сервисам.
package client

import "time"

// defaultTimeout — максимальное время на один gRPC-вызов.
// Используется всеми клиентами пакета (patient, appointment).
const defaultTimeout = 3 * time.Second
