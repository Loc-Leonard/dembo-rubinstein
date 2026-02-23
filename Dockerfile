FROM golang:1.21-alpine

# Создаем рабочую директорию
WORKDIR /app

# Копируем go.mod и go.sum
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект
COPY . .

# Собираем приложение
RUN go build -o main .

# Открываем порт
EXPOSE 8080

# Запускаем
CMD ["./main"]