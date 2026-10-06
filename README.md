# GoHospital

Микросервисная система для клиники (учебный проект).

## Сервисы
- **api-gateway** — HTTP-фронт (Gin)
- **patient-service** — gRPC + PostgreSQL
- **appointment-service** — NATS + PostgreSQL

## Технологии
Go 1.26, gRPC, Protobuf, NATS, PostgreSQL, Docker