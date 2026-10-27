# Шаг 1: Сборка
FROM golang:1.27 AS builder

WORKDIR /app

# Копируем go.work, go.work.sum и все go.mod для кэширования зависимостей
COPY go.work go.work.sum ./
COPY proto/go.mod proto/go.sum ./proto/
COPY patient-service/go.mod patient-service/go.sum ./patient-service/
COPY appointment-service/go.mod appointment-service/go.sum ./appointment-service/
COPY api-gateway/go.mod api-gateway/go.sum ./api-gateway/

# Скачиваем зависимости
RUN cd patient-service && go mod download

# Копируем исходники
COPY proto/ ./proto/
COPY patient-service/ ./patient-service/

# Собираем бинарник
RUN cd patient-service && CGO_ENABLED=0 GOOS=linux go build -o /patient-service ./cmd

# ---------------------------------------------------------
# Шаг 2: Финальный образ (начинается НОВЫЙ этап)
# ---------------------------------------------------------
FROM alpine:latest

# Устанавливаем базу часовых поясов
RUN apk add --no-cache tzdata

WORKDIR /root/

# Копируем бинарник из builder
COPY --from=builder /patient-service .

# Открываем порт
EXPOSE 50051

# Запускаем
CMD ["./patient-service"]