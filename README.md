# final_project_go_yandex

# ЛЮДИ, подождите пожалйста да звастра-послезаватра, честно не успеваю, был на хакатоне, войдите в положение. БУДУ вам очень благодварен. ЭТА надпись пропадет когда готово будет. tg: [@MamaKupiSnikes](https://t.me//MamaKupiSnikers)

## Описание

Система распределённого вычисления арифметических выражений состоит из двух сервисов:

* **Оркестратор**: принимает выражение через HTTP (порт 80), разбивает его на независимые задачи и отдаёт агентам по gRPC (порт 50051). Хранит данные в PostgreSQL.
* **Агент**: подключается к оркестратору по gRPC (AGENT\_URL из `.env`), запрашивает задачи, выполняет их и возвращает результаты.

## Требования

* Go 1.20+
* PostgreSQL
* .env-файл с переменными:

  ```dotenv
  # Orchestrator
  TIME_ADDITION_MS=3000
  TIME_SUBTRACTION_MS=3000
  TIME_MULTIPLICATIONS_MS=5000
  TIME_DIVISIONS_MS=5000

  # Agent
  COMPUTING_POWER=4
  AGENT_URL=localhost:50051

  # Postgres
  POSTGRES_DB=postgres_db
  POSTGRES_USER=postgres_user
  POSTGRES_PASSWORD=5050
  POSTGRES_HOST=localhost
  POSTGRES_PORT=5432

  # JWT
  JWT_TOKEN=your_super_secret_token
  ```

## Запуск

1. Клонировать репозиторий:

   ```bash
   git clone https://github.com/nais2008/final_project_go_yandex.git
   cd final_project_go_yandex
   ```

2. Установить зависимости и сгенерировать gRPC-код:

   ```bash
   go mod tidy
   protoc --go_out=. --go-grpc_out=. proto/sso/sso.proto
   ```

3. Запустить оркестратор:

   ```bash
   go run cmd/orchestrator/main.go
   ```

   * HTTP API: `http://localhost` (порт 80)
   * gRPC: `localhost:50051`
4. В другом терминале запустить агента:

   ```bash
   go run cmd/agent/main.go
   ```

## Примеры запросов

* Регистрация:

  ```bash
  curl -X POST http://localhost/api/v1/auth/register \
       -H "Content-Type: application/json" \
       -d '{"email":"user@example.com","username":"user1","password":"pass"}'
  ```

* Авторизация:

  ```bash
  curl -X POST http://localhost/api/v1/auth/login \
       -H "Content-Type: application/json" \
       -d '{"login":"user1","password":"pass"}'
  ```

* Отправка выражения:

  ```bash
  curl -X POST http://localhost/api/v1/expressions \
       -H "Content-Type: application/json" \
       -H "Authorization: Bearer <TOKEN>" \
       -d '{"expression":"2+3*4-1/5"}'
  ```

* Получение списка выражений:

  ```bash
  curl http://localhost/api/v1/expressions \
       -H "Authorization: Bearer <TOKEN>"
  ```

* Получение выражения по ID:

  ```bash
  curl http://localhost/api/v1/expressions/1 \
       -H "Authorization: Bearer <TOKEN>"
  ```

## Фронтенд

В `templates/base.html` реализована SPA-страница с использованием HTMX и Tailwind.
