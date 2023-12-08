# Используем официальный образ Golang
FROM golang:latest AS build

# Устанавливаем рабочую директорию
WORKDIR /OZON_test_task

# Копируем исходный код в образ
COPY . .

# Компилируем Go приложение
RUN go build -o main .

# Используем официальный образ PostgreSQL
FROM postgres:latest AS postgres

# Создаем базу данных и пользователя
ENV POSTGRES_DB ozon_db
ENV POSTGRES_USER user
ENV POSTGRES_PASSWORD 1234

# Используем официальный образ Golang во втором этапе
FROM golang:latest

# Устанавливаем рабочую директорию
WORKDIR /OZON_test_task

# Копируем скомпилированное приложение из предыдущего этапа
COPY --from=build /OZON_test_task/main .

COPY . .

RUN chmod +x /OZON_test_task/main
# Запускаем приложение
CMD ["go", "test", "./..."]
