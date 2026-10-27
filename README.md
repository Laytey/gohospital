# GoHospital

Микросервисная система для автоматизации записей к врачу в клинике.

Учебный проект: **3 микросервиса** на Go, связанные через **gRPC** и **NATS**, с изоляцией БД по сервисам.

## Стек

- **Go 1.27**
- **gRPC + Protobuf** — синхронное общение между сервисами
- **NATS** — асинхронные события (`appointment.created`, `appointment.canceled`)
- **PostgreSQL 17** — две изолированные БД (`patients_db`, `appointments_db`)
- **Gin** — HTTP-фронт
- **Docker + Docker Compose** — контейнеризация
- **go.work** — мультимодульный workspace

## Архитектура
 ┌─────────────────────┐
│ HTTP-клиент │
└──────────┬──────────┘
│ HTTP/JSON
▼
┌─────────────────────┐
│ api-gateway │ :8080
│ (Gin) │
└──┬───────────────┬──┘
│ gRPC │ gRPC
▼ ▼
┌─────────────────┐ ┌──────────────────────┐
│ patient-service │ │ appointment-service │
│ gRPC :50051 │◄─┤ gRPC :50052 │
│ │ │ │
│ patients_db │ │ appointments_db │
└─────────────────┘ └──────────┬───────────┘
│ publish
▼
┌─────────┐
│ NATS │ :4222
└────┬────┘
│ subscribe
▼
[Notification Worker]
(log-заглушка уведомлений)

## API

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/health` | Проверка живости сервиса |
| `POST` | `/patients` | Создать пациента |
| `GET` | `/patients/:id` | Получить пациента по ID |
| `POST` | `/appointments` | Создать запись к врачу |
| `GET` | `/appointments/:id` | Получить запись по ID |
| `POST` | `/appointments/:id/cancel` | Отменить запись |

## Запуск

**Требования:** Docker Desktop, ~1 ГБ свободного места.

```bash
git clone https://github.com/Laytey/gohospital.git
cd gohospital
docker compose up --build -d
Проверка:

bash
curl http://localhost:8080/health
# {"status":"ok"}
Остановка:

bash
docker compose down
Примеры запросов
Создать пациента:

bash
curl -X POST http://localhost:8080/patients \
  -H "Content-Type: application/json" \
  -d '{"name":"Иван Иванов","birthdate":"1990-01-15"}'
# {"id":"...","name":"Иван Иванов","birthdate":"1990-01-15"}
Создать запись:

bash
curl -X POST http://localhost:8080/appointments \
  -H "Content-Type: application/json" \
  -d '{"patient_id":"<UUID>","doctor":"Петров П.П.","time":"2026-11-05T14:00:00Z"}'
# {"id":"...","patientId":"<UUID>","doctor":"Петров П.П.","time":"...","status":"created"}
Отменить запись:

bash
curl -X POST http://localhost:8080/appointments/<UUID>/cancel
# {"id":"...","status":"canceled"}
Структура проекта:
text
gohospital/
├── go.work
├── docker-compose.yml
├── proto/                    # gRPC-контракты
│   ├── patient/
│   └── appointment/
├── patient-service/          # gRPC + PostgreSQL
├── appointment-service/      # gRPC + PostgreSQL + NATS
└── api-gateway/              # HTTP (Gin) → gRPC
