# Advanced Blog Management System

Дипломный backend-сервис на Go: REST API блога с JWT, PostgreSQL, комментариями и отложенной публикацией постов.

## Стек

- Go 1.21+
- Chi Router
- PostgreSQL 15
- JWT (`golang-jwt/jwt/v5`)
- bcrypt
- Docker / Docker Compose

## Архитектура

```
Handler → Service → Repository → PostgreSQL
```

Фоновый планировщик (`ticker` + worker pool + `context`) публикует посты с `publish_at <= now`.

## Быстрый старт

### 1. Конфигурация

```bash
cp .env.example .env
go mod download
```

### 2. База данных

```bash
docker compose up -d db
# подождите ~10–15 секунд
docker compose logs db
```

### 3. Запуск API

```bash
go run ./cmd/api
# или
make run
```

Сервис: `http://localhost:8080`

### 4. Полный Docker Compose

```bash
docker compose up --build
```

## Переменные окружения

| Переменная | Описание | По умолчанию |
|---|---|---|
| `DB_HOST` | Хост PostgreSQL | `localhost` |
| `DB_PORT` | Порт БД | `5432` |
| `DB_USER` | Пользователь | `postgres` |
| `DB_PASSWORD` | Пароль | `postgres` |
| `DB_NAME` | Имя БД | `blog_db` |
| `DB_SSLMODE` | SSL | `disable` |
| `JWT_SECRET` | Секрет JWT | обязателен |
| `SERVER_HOST` | Хост HTTP | `0.0.0.0` |
| `SERVER_PORT` | Порт HTTP | `8080` |
| `PUBLISHER_INTERVAL_SEC` | Интервал планировщика | `10` |
| `PUBLISHER_WORKERS` | Число воркеров | `4` |

Миграции из `migrations/` применяются автоматически при старте.

## API

### Публичные

| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/health` | Health check |
| POST | `/api/register` | Регистрация |
| POST | `/api/login` | Вход (JWT) |
| GET | `/api/posts` | Список постов (`limit`, `offset`) |
| GET | `/api/posts/{id}` | Пост по ID |
| GET | `/api/users/{id}/posts` | Посты автора |
| GET | `/api/posts/{postId}/comments` | Комментарии |

### Защищённые (`Authorization: Bearer <token>`)

| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/posts` | Создать пост |
| PUT | `/api/posts/{id}` | Обновить (только автор) |
| DELETE | `/api/posts/{id}` | Удалить (только автор) |
| POST | `/api/posts/{postId}/comments` | Добавить комментарий |
| PUT | `/api/comments/{id}` | Обновить комментарий |
| DELETE | `/api/comments/{id}` | Удалить комментарий |

### Модели

- **User**: `id`, `username`, `email`, `password` (bcrypt), timestamps
- **Post**: `id`, `title`, `content`, `author_id`, `status` (`draft`/`published`), `publish_at`, timestamps
- **Comment**: `id`, `content`, `post_id`, `author_id`, timestamps

Если при создании поста `publish_at` в будущем — статус `draft`, иначе `published`. Планировщик публикует due-посты конкурентно.

## Примеры curl

```bash
# Health
curl http://localhost:8080/api/health

# Регистрация
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"password123"}'

# Вход
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# Создать пост (подставьте TOKEN)
curl -X POST http://localhost:8080/api/posts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{"title":"My First Post","content":"Hello blog"}'

# Отложенная публикация
curl -X POST http://localhost:8080/api/posts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{"title":"Later","content":"Scheduled","publish_at":"2030-01-01T12:00:00Z"}'

# Список постов
curl "http://localhost:8080/api/posts?limit=10&offset=0"
```

## Тесты

```bash
go test ./... -v
go test ./... -race
go test ./... -cover
```

Покрыто в том числе:

- статус публикации при `publish_at`
- публикация due-постов
- worker pool планировщика
- JWT и bcrypt

## Структура

```
cmd/api/main.go
internal/handler/
internal/middleware/
internal/model/
internal/repository/
internal/service/
pkg/auth/
pkg/database/
migrations/
docker-compose.yml
.env.example
```

## Полезные команды

```bash
make db-up          # поднять PostgreSQL
make run            # запустить API
make test           # тесты
make docker-up      # app + db
docker compose exec db psql -U postgres -d blog_db
docker compose down -v
```
