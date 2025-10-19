# Subscriptions API (effective-mobile-test-task)

Сервис - papaloopalous.xyz:8085/(эндпоинт)

Swagger-документация - papaloopalous.xyz:8085/swagger/index.html

Сервис для управления пользовательскими подписками. Он позволяет:
- создавать подписки с помесячной ценой и длительностью;
- получать подписку по ID;
- обновлять стоимость и дату окончания;
- удалять подписку;
- получать список с фильтрами и курсорной пагинацией;
- считать суммарную стоимость по заданным фильтрам и периоду.

API документировано через Swagger и сопровождается unit и интеграционными тестами. В CI используется сборка, линт и тестирование.

## Отчёт о покрытии добавляется в артефакты каждого выполнения CI пайплайна

## Технологии
- Go 1.24, `gorilla/mux` - HTTP маршрутизация
- `pgx` - PostgreSQL драйвер и пул
- `zap` - структурированное логирование (JSON), логи в `./logs/logs.json`
- `viper` - конфиг
- `swaggo/swag` + `http-swagger` - Swagger-документация
- `tern` - миграции БД (SQL)
- Docker + Docker Compose - окружение для интеграционных тестов

## Структура проекта
- `app/`
  - `cmd/` - main-приложение
  - `api/handlers/` - HTTP-хендлеры
  - `api/router/` - настройка маршрутов
  - `api/response/` - единый формат ответа API
  - `internal/db/` - обёртка над `pgxpool` с логированием запросов
  - `internal/repo/` - интерфейс и реализация репозитория подписок
  - `internal/read_config/` - загрузка конфига (`./configs/config.yaml`)
  - `docs/` - сгенерированные Swagger-файлы
  - `tests/` - unit и интеграционные тесты
  - `util/` - константы (включая формат дат)
- `build/`
  - `compose.yml` - развёртка сервиса (сервис + Postgres)
  - `db/migrations/` - миграции `tern`
  - `config_template.txt`, `tern_conf_template.txt` - шаблоны для генерации
  - `Makefile` - цель `generate` для сборки конфигов

## Формат дат
Во всех запросах/ответах используется формат месяца: `MM-YYYY` (например, `01-2025`). Константа: `app/util/consts.go: DateFormat`.

## Старт (Docker Compose)
1) Установите Docker Compose

2) Создайте файл окружения в `/build` (пример как в CI):
```
cd build
cat > .env <<EOF
MAIN_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=test
DB_PASSWORD=test
DB_NAME=test
DB_SSLMODE=disable
SLOW_THRESHOLD=200ms
DB_POOL_MIN_CONNS=2
DB_POOL_MAX_CONNS=20
DB_POOL_MAX_CONN_LIFETIME=15m
DB_POOL_MAX_CONN_IDLE_TIME=5m
DB_POOL_HEALTH_CHECK_PERIOD=1m
DB_TIMEOUT=5s
EOF
```

3) Сгенерируйте конфиги в `/build` и поднимите сервис:
```
cd build
mkdir -p logs
mkdir -p configs
make generate
docker compose up -d
```

4) Примените миграции (установить tern и выполнить миграции):
```
go install github.com/jackc/tern/v2@latest
tern migrate --config ./db/tern.conf --migrations ./db/migrations
```

## Эндпоинты
- POST `/addSub` - создать подписку
  - Пример тела:
    ```json
    {
      "service_name": "Netflix",
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "monthly_fee": 499,
      "start_date": "10-2024",
      "num_months": 12
    }
    ```

- GET `/getSub?sub_id=<UUID>` - получить подписку по ID

- PUT `/updateSub` - обновить стоимость и дату окончания
  - Пример тела:
    ```json
    {
      "sub_id": "2f1e4e2a-1a7d-4e6b-a222-3cb3f92f0a11",
      "monthly_fee": 599,
      "end_date": "01-2025"
    }
    ```

- DELETE `/deleteSub?sub_id=<UUID>` - удалить подписку

- POST/GET `/listSubs` - список с фильтрами и курсором
  - Пример тела:
    ```json
    {
      "service_name": "Netflix",
      "user_id": "",
      "start_date": "01-2025",
      "end_date": "",
      "cursor": {
        "last_start": "01-2025",
        "last_id": "550e8400-e29b-41d4-a716-446655440000",
        "limit": 50
      }
    }
    ```

- POST/GET `/totalSubs` - суммарная стоимость по фильтрам и периоду
  - Тело запроса такое же, как у `/listSubs` (без курсора).

Единый формат ответа: `app/api/response/response.go`.

## Swagger
- UI: `/swagger/index.html`
- Регенерация (из каталога `app/`):
```
go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/main.go -o docs
```

## Тесты
- Unit-тесты:
```
cd app
go test ./tests -coverpkg=task_test/api/handlers,task_test/internal/repo -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html
```

- Интеграционные тесты (как в CI):
```
# подготовка окружения (см. старт)
cd build && make generate && docker compose up -d
go install github.com/jackc/tern/v2@latest && tern migrate --config ./db/tern.conf --migrations ./db/migrations
cd ../app && INTEGRATION=1 BASE_URL=http://localhost:8080 go test ./tests/integration_test.go -v
cd ../build && docker compose down -v
```

## CI Pipeline (GitHub Actions)
Файл: `.github/workflows/main.yml`
- Триггеры: push/PR в ветки `main`, `dev`, ручной запуск
- Матрица окружения: Go `${{ env.GO_VERSION }} = 1.24`, рабочая папка `./app`
- Джобы:
  - Build: `go mod download && go mod tidy`, затем `go build -v ./...`
  - Lint: `golangci-lint` (таймаут 5м)
  - Test (зависит от build и lint): unit-тесты с покрытием и публикацией `coverage.html`, сборка Docker image, подготовка `build/logs` и `build/configs`, `make generate`, поднятие `docker compose`, ожидание готовности Postgres, установка `tern`, применение миграций, прогон интеграционных тестов, `docker compose down -v` в конце

## P.S.
```
Независимо от результата, мне важно получить взгляд со стороны профессионалов.

Пожалуйста, поделитесь хотя бы кратким ревью - это поможет мне выстроить чёткие цели развития и понять, где расти дальше.
```
