# Subscription Service

**REST API для управления онлайн-подписками пользователей**

Проект реализует CRUDL-операции над подписками и предоставляет эндпоинт для подсчёта суммарной стоимости подписок за период.    
Логи осуществлены с помощью `zerolog`, конфигурация через `.env`. Запуск — через `docker compose`.


## Возможности

- Каждая запись содержит:
  - `service_name` — название сервиса, предоставляющего подписку
  - `price` — стоимость месячной подписки в рублях (целое число),
  - `user_id` — ID пользователя в формате UUID,
  - `start_date` — дата начала подписки (месяц и год, формат `MM-YYYY`),
  - `end_date` — дата окончания подписки (месяц и год, формат `MM-YYYY`). Опционально. Если не указана — автоматически `start_date + 30 дней`.
- Эндпоинт подсчёта суммы подписок за период (с фильтрами по `user_id` и `service_name`).
- СУБД — **PostgreSQL** (с миграциями).
- Конфигурационные данные вынесены в `.env`.
- Документация API — OpenAPI YAML (файл `docs/openapi.yaml`). Доступна по ссылке `http://localhost:8081/`
- Логи — `zerolog`.
- Запуск — `docker compose`.


## Технологии

- Go 1.24.4
- Docker & Docker Compose
- Postman

## Конфигурация (.env)

Для начала необходимо создать `.env` в корне проекта. Пример конфигурации указан в [.env.example](github.com/kweall/subscription-service/.env.example):

```
APP_PORT=8080
LOG_LEVEL=info

DB_HOST=db
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=subscriptions_db
DB_SSLMODE=disable
```

## Запуск (Docker Compose)

1. Собрать и поднять стек (в том числе контейнер миграций):

```bash
docker compose up --build
```

2. После старта приложение доступно: `http://localhost:8080` (порт берётся из `.env`).


### CRUDL подписок

- `POST /subscriptions` — создать подписку
    - Тело `JSON`:
    ```
    {
    "service_name": "Yandex Plus",
    "price": 199,
    "user_id": "7d9d8e22-bc1d-4dbe-9e4d-3c5dfedcb5b9",
    "start_date": "10-2025",
    "end_date": "11-2025" // опционально
    }
    ```
    - 201 Created — при правильных данных;
    - 400 Bad Request — при ошибке в данных;
- `GET /subscriptions` — получить список подписок
    - 200 OK — когда сервис в работе;
- `GET /subscriptions/{id}` — получить подписку по ID запроса (не пользователя)
    - 200 OK — если подписка найдена;
    - 400 Bad Request — при ошибке в данных;
    - 404 Not Found — если подписка не найдена;
- `PUT /subscriptions/{id}` — обновить подписку по ID запроса (не пользователя)
    - Тело `JSON`:
    ```
    {
    "service_name": "Yandex UltraPremium",
    "price": 499,
    "user_id": "7d9d8e22-bc1d-4dbe-9e4d-3c5dfedcb5b9",
    "start_date": "10-2025",
    "end_date": "11-2025" // опционально
    }
    ```
    - 200 OK — успешное изменение, если ID подписки уже есть в базе;
    - 400 Bad Request — при ошибке в данных (например, некорректная длина id);
    - 404 Not Found — если подписка не найдена;
- `DELETE /subscriptions/{id}` — удалить подписку по ID запроса (не пользователя)
    - 204 No Content — успешное удаление;
    - 400 Bad Request — при ошибке в данных (например, некорректная длина id);
    - 404 Not Found — если подписка не найдена;
- `GET /subscriptions/total` — подсчитать сумму подписок за период
    - Пример:
        `GET http://localhost:8080/subscriptions/total?from=2025-10-01&to=2025-11-01`
    - 200 OK — Вывод суммы
    - 400 Bad Request — при ошибке в данных;
    - 500 Internal Server Error

## Тесты

```bash
go test ./... -v
```

## Логи

Логирование реализовано через `zerolog`. Уровень логов настраивается через `LOG_LEVEL` в `.env`.

---

## Документация OpenAPI / Swagger

Файл: [docs/openapi.yaml](github.com/kweall/subscription-service/docs/openapi.yaml)
