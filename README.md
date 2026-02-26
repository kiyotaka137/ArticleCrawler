# ArticleCrawler

ArticleCrawler - сервис, который собирает статьи по ссылке и сохраняет их в PostgreSQL.
URL можно отправить через gRPC или HTTP. После обработки статья доступна в списке и в стриме новых материалов.

## Возможности

- gRPC API:
  - `SubmitUrl` - отправка URL в обработку
  - `GetArticle` - получение статьи по id
  - `ListArticles` - список статей с пагинацией
  - `StreamNewArticles` - поток новых статей
- HTTP API:
  - `GET /health`
  - `POST /submit`
  - `GET /stream` (SSE-прокси к gRPC stream)
- Обработка URL в несколько шагов:
  - не перегружает один и тот же сайт частыми запросами
  - при временной ошибке пробует запрос еще раз с паузой
  - вытаскивает заголовок и текст из HTML
  - добавляет служебные поля: короткое описание, язык, хеш, время чтения
- Хранение в PostgreSQL:
  - upsert по `url`
  - дедупликация по `content_hash`
  - лог попыток fetch

## Как это работает

1. Сервис получает URL.
2. Скачивает страницу.
3. Вытаскивает заголовок и текст.
4. Добавляет дополнительные данные (краткое описание, язык, время чтения).
5. Сохраняет результат в БД и отправляет событие подписчикам.

## Стек

- Go 1.24.6
- gRPC + protobuf
- Gin (HTTP/SSE)
- PostgreSQL 15
- Docker / Docker Compose

## Структура проекта

- `cmd/main.go` - запуск сервиса
- `cmd/e2e/main.go` - e2e проверка (submit + проверка записи в БД)
- `cmd/load_test/main.go` - простой нагрузочный RPC-тест
- `internal/pipeline/*` - этапы пайплайна
- `internal/server/server.go` - gRPC сервер
- `internal/db/*` - репозиторий и миграции
- `internal/config/config.go` - загрузка YAML-конфига
- `pkg/proto/crawler.proto` - контракт API

## Запуск

### Вариант 1: Docker Compose

```bash
docker compose up -d --build
```

По умолчанию:

- gRPC: `localhost:50051`
- HTTP: `localhost:8080`
- PostgreSQL: `localhost:5434`

### Вариант 2: Локально

1. Поднять PostgreSQL.
2. Применить миграцию `internal/db/migrations/001_create_tables.sql`.
3. Настроить `config.yaml` (или передать свой файл через `-config`).
4. Запустить:

```bash
go run ./cmd -config config.yaml
```

## Конфиг

Пример `config.yaml`:

```yaml
server:
  grpc_addr: ":50051"
  http_addr: ":8080"
pipeline:
  fetch_workers: 4
  parse_workers: 4
  enrich_workers: 2
  store_workers: 2
rate_limit:
  default_rps: 2
  burst: 5
database:
  url: "postgres://crawler:crawlerpass@db:5432/crawler?sslmode=disable"
backoff:
  base_seconds: 1
  max_retries: 3
```

## Тесты и результаты

Проверка, что URL доходит до БД:

```bash
go run ./cmd/e2e
```

Тест с метриками (скорость и задержка):

```bash
go run ./cmd/load_test
```

RPS считался на gRPC-ручке `SubmitUrl` (`localhost:50051`).
Это самая показательная ручка для замера производительности на входе, потому что через нее в сервис поступают все новые URL на обработку.

Фактический результат прогона метрик:

```text
Total requests: 100
Total time: 19.903669ms
Average latency per RPC: 1.929949ms
Max latency: 8.485107ms
RPS: 5024.20
```
