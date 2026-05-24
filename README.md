# Org API

REST API для управления организационной структурой компании:
дерево подразделений и сотрудники.

## Стек

| Компонент   | Технология       |
|-------------|------------------|
| Язык        | Go 1.25          |
| HTTP-сервер | net/http         |
| ORM         | GORM             |
| База данных | PostgreSQL 16    |
| Миграции    | goose            |
| Контейнеры  | Docker + Compose |

---

## Быстрый старт

### 1. Клонировать репозиторий

```bash
git clone https://github.com/merclamp/org-api.git
cd org-api
```

### 2. Создать .env (опционально)

Все переменные уже имеют значения по умолчанию в `docker-compose.yml`,
поэтому `.env` нужен только если хочешь переопределить что-то.

```bash
cp .env.example .env
```

### 3. Запустить

```bash
docker compose up --build
```

- API: `http://localhost:8080`
- PostgreSQL: `localhost:5432`
- Миграции применяются **автоматически** при старте

### 4. Проверить

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

---

## Остановка

```bash
# Остановить контейнеры
docker compose down

# Остановить и удалить данные БД
docker compose down -v
```

---

## Запуск без Docker (локальная разработка)

```bash
# Поднять только postgres
docker compose up postgres -d

# Скопировать конфиг
cp .env.example .env

# Запустить приложение
go run ./cmd/api
```

---

## Структура проекта

```
org-api/
├── cmd/
│   └── api/
│       └── main.go              — точка входа, сборка всех слоёв
├── internal/
│   ├── config/
│   │   └── config.go            — конфигурация из переменных окружения
│   ├── domain/
│   │   ├── department.go        — модель подразделения
│   │   ├── employee.go          — модель сотрудника
│   │   └── errors.go            — доменные ошибки
│   ├── repository/
│   │   ├── department_repo.go   — CRUD подразделений (GORM)
│   │   └── employee_repo.go     — CRUD сотрудников (GORM)
│   ├── service/
│   │   ├── department_svc.go    — бизнес-логика подразделений
│   │   └── employee_svc.go      — бизнес-логика сотрудников
│   ├── handler/
│   │   ├── department.go        — HTTP-хендлеры подразделений
│   │   ├── employee.go          — HTTP-хендлеры сотрудников
│   │   ├── router.go            — регистрация маршрутов
│   │   ├── response.go          — helpers для JSON-ответов
│   │   ├── decode.go            — декодирование тела запроса
│   │   └── params.go            — парсинг query и path параметров
│   └── middleware/
│       └── logger.go            — логирование HTTP-запросов
├── migrations/
│   ├── 001_create_departments.sql
│   └── 002_create_employees.sql
├── Dockerfile
├── docker-compose.yml
├── .env.example
└── README.md
```

---

## Переменные окружения

| Переменная      | По умолчанию | Описание                    |
|-----------------|--------------|-----------------------------|
| `SERVER_PORT`   | `8080`       | Порт HTTP-сервера           |
| `DB_HOST`       | `localhost`  | Хост PostgreSQL             |
| `DB_PORT`       | `5432`       | Порт PostgreSQL             |
| `DB_USER`       | `postgres`   | Пользователь БД             |
| `DB_PASSWORD`   | `postgres`   | Пароль БД                   |
| `DB_NAME`       | `orgapi`     | Имя базы данных             |
| `DB_SSLMODE`    | `disable`    | SSL-режим PostgreSQL        |

---

## API Reference

### Подразделения

#### POST /departments/
Создать подразделение.

```bash
curl -X POST http://localhost:8080/departments/ \
  -H "Content-Type: application/json" \
  -d '{"name": "Engineering", "parent_id": 1}'
```

Тело запроса:

| Поле        | Тип        | Обязательно | Описание                        |
|-------------|------------|-------------|---------------------------------|
| `name`      | `string`   | да          | Название (1..200 символов)      |
| `parent_id` | `int/null` | нет         | ID родительского подразделения  |

Ответ `201 Created`:

```json
{
  "id": 2,
  "name": "Engineering",
  "parent_id": 1,
  "created_at": "2025-01-15T10:00:00Z"
}
```

---

#### GET /departments/{id}
Получить подразделение с деревом и сотрудниками.

```bash
curl "http://localhost:8080/departments/1?depth=3&include_employees=true"
```

Query-параметры:

| Параметр            | Тип    | По умолчанию | Описание                          |
|---------------------|--------|--------------|-----------------------------------|
| `depth`             | `int`  | `1`          | Глубина дерева (1..5)             |
| `include_employees` | `bool` | `true`       | Включать ли сотрудников в ответ   |

Ответ `200 OK`:

```json
{
  "id": 1,
  "name": "Company",
  "parent_id": null,
  "created_at": "2025-01-15T10:00:00Z",
  "employees": [],
  "children": [
    {
      "id": 2,
      "name": "Engineering",
      "parent_id": 1,
      "created_at": "2025-01-15T10:01:00Z",
      "employees": [],
      "children": [
        {
          "id": 4,
          "name": "Backend",
          "parent_id": 2,
          "created_at": "2025-01-15T10:02:00Z",
          "employees": [
            {
              "id": 1,
              "department_id": 4,
              "full_name": "Ivan Ivanov",
              "position": "Go Developer",
              "hired_at": "2024-03-01T00:00:00Z",
              "created_at": "2025-01-15T10:03:00Z"
            }
          ]
        }
      ]
    }
  ]
}
```

---

#### PATCH /departments/{id}
Переименовать или переместить подразделение.

```bash
# Переименовать
curl -X PATCH http://localhost:8080/departments/2 \
  -H "Content-Type: application/json" \
  -d '{"name": "Tech"}'

# Переместить в другой parent
curl -X PATCH http://localhost:8080/departments/2 \
  -H "Content-Type: application/json" \
  -d '{"parent_id": 3}'

# Сделать корневым (убрать parent)
curl -X PATCH http://localhost:8080/departments/2 \
  -H "Content-Type: application/json" \
  -d '{"parent_id": null}'
```

Тело запроса (все поля опциональны):

| Поле        | Тип        | Описание                                          |
|-------------|------------|---------------------------------------------------|
| `name`      | `string`   | Новое название                                    |
| `parent_id` | `int/null` | Новый родитель; `null` — сделать корневым         |

Ответ `200 OK` — обновлённое подразделение.

---

#### DELETE /departments/{id}
Удалить подразделение.

```bash
# Каскадное удаление (дети + сотрудники)
curl -X DELETE "http://localhost:8080/departments/2?mode=cascade"

# Перевод сотрудников в другое подразделение перед удалением
curl -X DELETE "http://localhost:8080/departments/2?mode=reassign&reassign_to_department_id=1"
```

Query-параметры:

| Параметр                     | Тип     | Описание                                            |
|------------------------------|---------|-----------------------------------------------------|
| `mode`                       | `string`| `cascade` или `reassign`                            |
| `reassign_to_department_id`  | `int`   | Обязателен при `mode=reassign`                      |

Ответ `204 No Content`.

---

### Сотрудники

#### POST /departments/{id}/employees/
Создать сотрудника в подразделении.

```bash
curl -X POST http://localhost:8080/departments/4/employees/ \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Ivan Ivanov",
    "position":  "Go Developer",
    "hired_at":  "2024-03-01"
  }'
```

Тело запроса:

| Поле        | Тип          | Обязательно | Описание                        |
|-------------|--------------|-------------|---------------------------------|
| `full_name` | `string`     | да          | ФИО (1..200 символов)           |
| `position`  | `string`     | да          | Должность (1..200 символов)     |
| `hired_at`  | `string/null`| нет         | Дата найма в формате YYYY-MM-DD |

Ответ `201 Created`:

```json
{
  "id": 1,
  "department_id": 4,
  "full_name": "Ivan Ivanov",
  "position": "Go Developer",
  "hired_at": "2024-03-01T00:00:00Z",
  "created_at": "2025-01-15T10:03:00Z"
}
```

---

## Коды ответов

| Код  | Описание                                                  |
|------|-----------------------------------------------------------|
| `200`| Успешный запрос                                           |
| `201`| Ресурс создан                                             |
| `204`| Успешное удаление (тело пустое)                           |
| `400`| Ошибка валидации (пустое поле, неверный формат)           |
| `404`| Ресурс не найден                                          |
| `409`| Конфликт (дублирование имени, цикл в дереве)              |
| `500`| Внутренняя ошибка сервера                                 |

---

## Бизнес-правила

| Правило                                  | Поведение         |
|------------------------------------------|-------------------|
| Дублирование имени в одном parent        | `409 Conflict`    |
| Несуществующий parent при создании       | `404 Not Found`   |
| Создание сотрудника в несущ. отделе      | `404 Not Found`   |
| Самоссылка (`parent_id == id`)           | `409 Conflict`    |
| Перемещение подразделения в своё поддерево | `409 Conflict`  |
| Пустое имя / должность / ФИО            | `400 Bad Request` |
| Длина поля > 200 символов               | `400 Bad Request` |
| `mode=reassign` без `reassign_to_id`     | `400 Bad Request` |
| Удаление `cascade`                       | Рекурсивно удаляет дочерние подразделения и сотрудников |
| Удаление `reassign`                      | Сотрудники переводятся, дочерние подразделения удаляются каскадом |

---

## Запуск тестов

```bash
go test ./... -v
```

С покрытием:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```