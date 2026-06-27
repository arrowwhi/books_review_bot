# CLAUDE.md — Книжный дневник (Book Diary Bot)

Этот файл — главный источник истины для всего проекта. Перед любыми изменениями прочитай его целиком.

---

## 1. Цель проекта

Telegram-бот для учёта прочитанных книг, рецензий и списка желаемого. Поддерживает нескольких пользователей: у каждого своя библиотека, изолированная по `user_id`. Никакой публичной регистрации нет — бот просто работает с любым, кто его нашёл, и хранит данные отдельно для каждого.

---

## 2. Технический стек

| Компонент        | Инструмент / Библиотека                                  |
|------------------|----------------------------------------------------------|
| Язык             | Go 1.21+                                                 |
| Telegram API     | `github.com/go-telegram-bot-api/telegram-bot-api/v5`    |
| База данных      | PostgreSQL 15+                                           |
| Драйвер БД       | `github.com/jackc/pgx/v5` (pgxpool)                     |
| Миграции         | `github.com/pressly/goose/v3`                            |
| Конфиг           | `github.com/joho/godotenv` + переменные окружения        |
| Логирование      | `go.uber.org/zap` (структурированные JSON-логи)          |
| Моки             | `go.uber.org/mock` + `mockgen`                           |
| Тесты            | `github.com/stretchr/testify` (assert/require)           |
| Claude API       | `github.com/anthropics/anthropic-sdk-go` (рекомендации) |
| Сборка           | `Makefile`                                               |
| Оркестрация      | `docker-compose.yml` (PostgreSQL + bot)                  |
| Hot reload (dev) | `air` (`github.com/air-verse/air`)                       |
| Хостинг          | VPS, systemd-сервис                                      |

---

## 3. Архитектура — Clean Architecture

Зависимости направлены строго внутрь: Handler → Service → Repository → Domain.

```
cmd/bot/
└── main.go                    # точка входа: сборка зависимостей (DI вручную), запуск бота

internal/
├── config/
│   └── config.go              # загрузка .env, структура Config

├── domain/                    # внутренний слой — только сущности и ошибки
│   ├── book.go                # Book, BookStatus, Emotion
│   ├── genre.go               # Genre
│   ├── stats.go               # Stats, AspectStats
│   └── errors.go              # ErrNotFound, ErrForbidden и т.д.

├── repository/                # интерфейсы репозиториев + их моки (в том же пакете)
│   ├── book.go                # BookRepository interface  +  //go:generate
│   ├── mock_book.go           # сгенерированный мок (package repository)
│   ├── genre.go               # GenreRepository interface
│   ├── mock_genre.go          # сгенерированный мок
│   ├── reminder.go            # ReminderRepository interface
│   ├── mock_reminder.go       # сгенерированный мок
│   └── postgres/              # реализации на pgx
│       ├── db.go              # pgxpool.Pool — инициализация соединения
│       ├── book.go            # *BookRepo implements BookRepository
│       ├── genre.go           # *GenreRepo implements GenreRepository
│       └── reminder.go        # *ReminderRepo implements ReminderRepository

├── service/                   # бизнес-логика + интерфейсы для хендлеров + моки
│   ├── interfaces.go          # BookService, GenreService, StatsService, RecommendService, ReminderService  +  //go:generate
│   ├── mock_interfaces.go     # сгенерированные моки всех сервисных интерфейсов (package service)
│   ├── book.go                # *BookSvc
│   ├── genre.go               # *GenreSvc
│   ├── stats.go               # *StatsSvc
│   ├── recommend.go           # *RecommendSvc (вызывает Claude API)
│   └── reminder.go            # *ReminderSvc + фоновая горутина проверки

├── client/
│   ├── openlibrary/
│   │   └── client.go          # HTTP-клиент Open Library API
│   └── claude/
│       └── client.go          # HTTP-клиент Anthropic Claude API

├── bot/
│   ├── bot.go                 # инициализация бота, выбор режима webhook/polling, запуск
│   ├── middleware.go          # логирование входящих апдейтов (user_id, command, duration)
│   ├── router.go              # маршрутизация Update → нужный Handler
│   │
│   ├── session/
│   │   ├── session.go         # Session struct, State constants, Draft struct
│   │   └── manager.go         # SessionManager (in-memory, sync.Map)
│   │
│   └── handler/
│       ├── common.go          # общие утилиты: escapeMarkdown, buildKeyboard, sendCard и т.д.
│       ├── add.go             # /add — пошаговый диалог добавления книги
│       ├── want.go            # /want — добавить в вишлист
│       ├── wishlist.go        # /wishlist — просмотр вишлиста
│       ├── library.go         # /library — список прочитанных
│       ├── search.go          # /search — поиск по своим книгам
│       ├── stats.go           # /stats — статистика
│       ├── recommend.go       # /recommend — рекомендации через Claude
│       ├── remind.go          # /remind — настройка напоминаний
│       └── help.go            # /help, /start

migrations/
└── 001_initial.sql            # genres, books, reminders

Makefile
docker-compose.yml             # postgres + bot (dev)
Dockerfile                     # multi-stage сборка для prod
.air.toml                      # конфиг hot reload (air)
.env.example
CLAUDE.md
go.mod
go.sum
```

### Правило расположения моков

Моки живут **в том же пакете**, что и интерфейс, рядом с файлом интерфейса:

- `internal/repository/book.go` → `internal/repository/mock_book.go` (package `repository`)
- `internal/service/interfaces.go` → `internal/service/mock_interfaces.go` (package `service`)

Директива генерации ставится в файл с интерфейсом:
```go
//go:generate mockgen -source=book.go -destination=mock_book.go -package=repository
```

Для сервисных интерфейсов — все в одном файле:
```go
//go:generate mockgen -source=interfaces.go -destination=mock_interfaces.go -package=service
```

---

## 4. Схема базы данных

### Таблица `genres`

```sql
CREATE TABLE genres (
    id         SERIAL PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE
);
```

Предустановленные жанры (is_default = true):
- Фантастика
- Детектив
- Историческая
- Нон-фикшн

### Таблица `books`

```sql
CREATE TABLE books (
    id             SERIAL PRIMARY KEY,
    user_id        BIGINT NOT NULL,
    title          TEXT NOT NULL,
    author         TEXT,
    genre_id       INTEGER REFERENCES genres(id) ON DELETE SET NULL,
    ol_key         TEXT,
    cover_url      TEXT,
    status         TEXT NOT NULL CHECK (status IN ('read', 'wishlist')),
    rating         SMALLINT CHECK (rating BETWEEN 1 AND 5),
    emotion        TEXT CHECK (emotion IN ('love','like','neutral','dislike','mixed')),
    aspect_plot    SMALLINT CHECK (aspect_plot BETWEEN 1 AND 10),
    aspect_chars   SMALLINT CHECK (aspect_chars BETWEEN 1 AND 10),
    aspect_atmo    SMALLINT CHECK (aspect_atmo BETWEEN 1 AND 10),
    aspect_ideas   SMALLINT CHECK (aspect_ideas BETWEEN 1 AND 10),
    aspect_style   SMALLINT CHECK (aspect_style BETWEEN 1 AND 10),
    aspect_tempo   SMALLINT CHECK (aspect_tempo BETWEEN 1 AND 10),
    liked_text     TEXT,
    disliked_text  TEXT,
    insight_text   TEXT,
    recommend      BOOLEAN,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at    TIMESTAMPTZ
);

CREATE INDEX idx_books_user_id ON books(user_id);
CREATE INDEX idx_books_status  ON books(user_id, status);
```

### Таблица `reminders`

```sql
CREATE TABLE reminders (
    user_id        BIGINT PRIMARY KEY,
    interval_days  INTEGER NOT NULL DEFAULT 14,
    last_sent_at   TIMESTAMPTZ,
    enabled        BOOLEAN NOT NULL DEFAULT TRUE
);
```

---

## 5. Команды бота

| Команда           | Описание                                         |
|-------------------|--------------------------------------------------|
| `/start`          | Приветствие + список команд                      |
| `/help`           | Список всех команд                               |
| `/add`            | Пошаговый диалог добавления прочитанной книги    |
| `/want`           | Добавить книгу в «хочу прочитать»                |
| `/wishlist`       | Просмотр вишлиста с пагинацией                   |
| `/library`        | Все прочитанные книги с пагинацией               |
| `/search <текст>` | Поиск по названию/автору среди своих книг        |
| `/stats`          | Статистика и аналитика                           |
| `/recommend`      | Рекомендации на основе вкусов через Claude API   |
| `/remind`         | Настроить напоминание о чтении                   |

---

## 6. Внешние API

### Open Library API

- **Поиск книг:** `GET https://openlibrary.org/search.json?title={query}&limit=3`
- **Обложка:** `https://covers.openlibrary.org/b/id/{cover_id}-M.jpg`
- Используется в шаге 2 команды `/add` для автозаполнения автора и обложки
- Клиент: `internal/client/openlibrary/client.go`
- Интерфейс: `OLClient` с методом `Search(ctx, title) ([]OLBook, error)`

### Anthropic Claude API

- **Endpoint:** `https://api.anthropic.com/v1/messages`
- **Модель:** `claude-sonnet-4-6` (или актуальная на момент деплоя)
- **Назначение:** Команда `/recommend` — персонализированные рекомендации книг
- **Ключ:** переменная окружения `ANTHROPIC_API_KEY`
- Клиент: `internal/client/claude/client.go`
- Интерфейс: `ClaudeClient` с методом `Complete(ctx, prompt) (string, error)`
- Промпт строится в `internal/service/recommend.go` на основе жанров и рейтингов пользователя

---

## 7. FSM — состояния сессии (команда `/add`)

Состояния хранятся в памяти (`SessionManager`, `sync.Map`). При рестарте бота незавершённые диалоги сбрасываются — это приемлемо для личного бота.

```
StateIdle
  └─/add──► StateAddTitle
                └─(ввод текста)──► [поиск OL] ──► StateAddSearchResult
                                                        ├─(выбор варианта)──► StateAddRating  (автор/обложка взяты из OL)
                                                        └─(«Не то» / «Пропустить»)──► StateAddAuthor
                                                                                            └─(ввод/пропуск)──► StateAddGenre
                                                                                                                    ├─(стандартный жанр)──► StateAddRating
                                                                                                                    ├─(«Другое» → показ кастомных)──► StateAddGenre
                                                                                                                    └─(«Новый жанр»)──► StateAddCustomGenre
                                                                                                                                            └─(ввод)──► StateAddRating
StateAddRating
  └─(кнопка ★1–★5)──► StateAddEmotion
StateAddEmotion
  └─(кнопка)──► StateAddAspectPlot
StateAddAspectPlot  → StateAddAspectChars → StateAddAspectAtmo →
StateAddAspectIdeas → StateAddAspectStyle → StateAddAspectTempo
  └─(кнопки 1–10 или «Пропустить»)──► каждый следующий аспект
StateAddAspectTempo
  └─► StateAddLiked
StateAddLiked
  └─(текст или «Пропустить»)──► StateAddDisliked
StateAddDisliked
  └─(текст или «Пропустить»)──► StateAddInsight
StateAddInsight
  └─(текст или «Пропустить»)──► StateAddRecommend
StateAddRecommend
  └─(Да/Нет)──► [сохранение] ──► StateIdle  (показ итоговой карточки)
```

На любом шаге «Отмена» → `StateIdle`, очистка черновика.

### Состояния `/want`

```
StateIdle ──/want──► StateWantTitle
  └─(ввод)──► StateWantAuthor
                └─(ввод или «Пропустить»)──► [сохранение] ──► StateIdle
```

### Состояния редактирования

```
StateIdle ──(кнопка Редактировать)──► StateEditField
  └─(ввод нового значения)──► [обновление] ──► StateIdle
```

---

## 8. Формат Callback Data

Все callback_data не превышают 64 байта. Разделитель — двоеточие (`:`).

| Callback               | Описание                                      |
|------------------------|-----------------------------------------------|
| `a:s:{idx}`            | Выбрать результат поиска OL (idx = 0,1,2)    |
| `a:s:skip`             | Не то / пропустить поиск OL                  |
| `a:au:skip`            | Пропустить автора                             |
| `a:g:{id}`             | Выбрать жанр по ID                            |
| `a:g:oth`              | Показать другие/кастомные жанры              |
| `a:g:new`              | Ввести новый жанр                             |
| `a:r:{1-5}`            | Рейтинг                                       |
| `a:e:{emotion}`        | Ощущение (love/like/neutral/dislike/mixed)    |
| `a:ap:{1-10}`          | Аспект: сюжет                                 |
| `a:ac:{1-10}`          | Аспект: персонажи                             |
| `a:aa:{1-10}`          | Аспект: атмосфера                             |
| `a:ai:{1-10}`          | Аспект: идеи                                  |
| `a:as:{1-10}`          | Аспект: стиль                                 |
| `a:at:{1-10}`          | Аспект: темп                                  |
| `a:asp:skip`           | Пропустить все аспекты                        |
| `a:lk:skip`            | Пропустить «что зацепило»                     |
| `a:dl:skip`            | Пропустить «что не понравилось»               |
| `a:in:skip`            | Пропустить «мысль/инсайт»                     |
| `a:rec:yes`            | Порекомендовал бы: да                         |
| `a:rec:no`             | Порекомендовал бы: нет                        |
| `a:cancel`             | Отмена добавления                             |
| `l:p:{page}`           | Страница библиотеки                           |
| `l:v:{id}`             | Просмотр книги                                |
| `w:p:{page}`           | Страница вишлиста                             |
| `w:v:{id}`             | Просмотр книги из вишлиста                    |
| `w:r:{id}`             | Перенести из вишлиста в прочитанное           |
| `s:p:{page}`           | Страница поиска                               |
| `s:v:{id}`             | Просмотр из поиска                            |
| `b:e:{id}`             | Редактировать книгу                           |
| `b:ef:{id}:{field}`    | Выбрать поле для редактирования               |
| `b:d:{id}`             | Удалить книгу (запрос подтверждения)         |
| `b:dc:{id}`            | Подтвердить удаление                          |
| `b:back:{id}`          | Назад к карточке книги                        |
| `rm:2w`                | Напоминание раз в 2 недели                    |
| `rm:1m`                | Напоминание раз в месяц                       |
| `rm:off`               | Выключить напоминание                         |

Поля для редактирования (`{field}`): `title`, `author`, `genre`, `rating`, `emotion`, `liked`, `disliked`, `insight`, `rec`

---

## 9. Переменные окружения (`.env`)

```env
# Telegram
BOT_TOKEN=          # токен от @BotFather

# Режим работы бота
BOT_MODE=webhook    # webhook | polling

# Webhook (нужен только при BOT_MODE=webhook)
WEBHOOK_URL=        # https://yourdomain.com
WEBHOOK_PORT=8443   # порт для входящих запросов от Telegram
WEBHOOK_PATH=/tg    # путь (без слеша в конце)

# PostgreSQL
DATABASE_URL=postgres://user:pass@localhost:5432/books_bot?sslmode=disable

# APIs
ANTHROPIC_API_KEY=  # ключ для Claude API

# Логирование
LOG_LEVEL=info      # debug | info | warn | error
```

---

## 10. Принципы кодирования

### Архитектура

- **Dependency Inversion:** все слои зависят от интерфейсов, не от конкретных реализаций
- **DI вручную:** никаких wire/fx — зависимости собираются в `main.go` явно
- **Интерфейсы определяются там, где используются:** `service/interfaces.go` — для хендлеров; `repository/*.go` — для сервисов
- Каждый пакет экспортирует только то, что нужно снаружи

### Обработка ошибок

- Ошибки оборачиваются через `fmt.Errorf("...: %w", err)` с контекстом
- Доменные ошибки (`domain/errors.go`) используются для разграничения не-найдено vs системные ошибки
- Хендлер логирует ошибку и отправляет пользователю дружелюбное сообщение, не паникуя
- `panic` запрещён, кроме `main.go` при фатальной ошибке инициализации

### Логирование (zap)

- Логи в JSON-формате для продакшена, human-readable для разработки (`LOG_LEVEL=debug`)
- Логируем: входящие апдейты (command + user_id), все ошибки с контекстом, вызовы внешних API (без токенов)
- НЕ логируем: тексты сообщений пользователя (приватность), credentials
- Поля: `user_id`, `command`, `state`, `book_id`, `error`, `duration`

### Тесты

- Тестируем: сервисный слой (бизнес-логика), хелперы форматирования, клиент Open Library
- НЕ тестируем: хендлеры напрямую (слишком много mock boilerplate), main.go
- Моки генерируются через `mockgen` из интерфейсов, хранятся в `mocks/`
- Команда генерации: `make mock` (запускает `go generate ./...`)
- Тест-файлы рядом с тестируемым кодом: `book_test.go` рядом с `book.go`

### Telegram UX

- ParseMode: `MarkdownV2` везде. Все пользовательские строки проходят через `escapeMarkdown()` из `handler/common.go`
- Пагинация: 5 элементов на страницу
- Каждый шаг диалога — отдельное сообщение (не редактируем предыдущее без нужды)
- После каждого callback_query обязательно отвечаем `AnswerCallbackQuery`
- Итоговая карточка показывается после сохранения книги

### Мультипользовательность

- Бот доступен любому пользователю Telegram — ограничений по доступу нет
- Все данные изолированы по `user_id` из Telegram: каждый видит только свои книги, свою статистику, свои напоминания
- `user_id` берётся из `update.Message.From.ID` / `update.CallbackQuery.From.ID` — это гарантирует изоляцию
- Никакой регистрации/авторизации нет, первое сообщение `/start` уже создаёт контекст пользователя

---

## 11. Makefile — основные команды

Сборка и запуск проекта — только через `make`. Никаких голых `go run` в документации.

```makefile
make dev          # docker compose --profile dev up  (postgres + bot с air внутри)
make prod         # docker compose --profile prod up -d
make build        # собрать бинарник в bin/bot локально
make test         # go test ./...
make mock         # go generate ./... (регенерация всех моков)
make migrate-up   # goose -dir migrations postgres $DATABASE_URL up
make migrate-down # goose -dir migrations postgres $DATABASE_URL down
make lint         # golangci-lint run
make down         # docker compose down
```

---

## 12. Запуск и деплой

### Локальная разработка с Docker Compose

```bash
cp .env.example .env
# заполнить BOT_TOKEN, ANTHROPIC_API_KEY и т.д.
make docker-dev       # docker compose --profile dev up (postgres + bot с air внутри)
make migrate-up       # накатить миграции
```

Air запускается **внутри контейнера** `bot`. Исходники примонтированы как volume, поэтому любое изменение `.go`-файла на хосте сразу вызывает пересборку и перезапуск внутри контейнера.

### Конфигурация Air (`.air.toml`)

```toml
[build]
  cmd = "go build -o ./tmp/bot ./cmd/bot"
  bin = "./tmp/bot"
  include_ext = ["go"]
  exclude_dir = ["tmp", "vendor"]

[log]
  time = true
```

### Docker Compose (`docker-compose.yml`)

Два профиля: `dev` (air внутри контейнера) и `prod` (скомпилированный бинарник).

```yaml
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: books_bot
      POSTGRES_USER: books
      POSTGRES_PASSWORD: books
    ports:
      - "5432:5432"
    volumes:
      - pg_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U books"]
      interval: 5s
      timeout: 5s
      retries: 5

  bot:
    build:
      context: .
      target: dev          # dev-стадия Dockerfile, содержит air
    env_file: .env
    volumes:
      - .:/app             # монтируем исходники — air видит изменения
      - go_cache:/root/go  # кэш модулей между перезапусками
    depends_on:
      postgres:
        condition: service_healthy
    profiles: [dev]

  bot-prod:
    build:
      context: .
      target: prod
    env_file: .env
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped
    profiles: [prod]

volumes:
  pg_data:
  go_cache:
```

```bash
# dev: бот с hot reload внутри контейнера
docker compose --profile dev up

# prod: скомпилированный бинарник
docker compose --profile prod up -d
```

### Dockerfile (multi-stage с dev и prod стадиями)

```dockerfile
# ── base: общие зависимости ──────────────────────────────────────────
FROM golang:1.21-alpine AS base
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# ── dev: air для hot reload ──────────────────────────────────────────
FROM base AS dev
RUN go install github.com/air-verse/air@latest
COPY . .
CMD ["air", "-c", ".air.toml"]

# ── builder: компиляция ──────────────────────────────────────────────
FROM base AS builder
COPY . .
RUN go build -o bin/bot ./cmd/bot

# ── prod: минимальный образ ──────────────────────────────────────────
FROM alpine:3.19 AS prod
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/bot ./bot
ENTRYPOINT ["./bot"]
```

### Продакшн (VPS + systemd)

Файл `/etc/systemd/system/books-bot.service`:

```ini
[Unit]
Description=Books Review Bot
After=network.target postgresql.service

[Service]
Type=simple
User=books
WorkingDirectory=/opt/books-bot
EnvironmentFile=/opt/books-bot/.env
ExecStart=/opt/books-bot/bin/bot
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

```bash
systemctl enable books-bot
systemctl start books-bot
journalctl -u books-bot -f
```

---

## 13. Карточка книги — формат вывода

```
📚 *Название книги*
👤 Автор: Имя Фамилия
🏷 Жанр: Фантастика
⭐️ Рейтинг: ★★★★☆ \(4/5\)
😍 Ощущение: Восторг

📊 *Аспекты:*
• Сюжет: 9/10
• Персонажи: 8/10
• Атмосфера: 9/10
• Идеи: 7/10
• Язык: 8/10
• Темп: 6/10

💬 Зацепило: текст\.\.\.
😞 Не понравилось: текст\.\.\.
💡 Инсайт: текст\.\.\.

👍 Порекомендую: Да

📅 Прочитано: 27 июня 2026
```

---

## 14. Карточка статистики — формат вывода

```
📊 *Ваша статистика*

📚 Прочитано: 42 книги \(15 в 2026 году\)
⭐️ Средний рейтинг: 4\.2

📁 *По жанрам:*
• Фантастика — 15 книг, ср\. 4\.5 ⭐
• Нон\-фикшн — 10 книг, ср\. 3\.8 ⭐

🏆 *Топ\-3 книги:*
1\. Название — ★★★★★
2\. Название — ★★★★☆
3\. Название — ★★★★☆

💪 Любимый аспект: Атмосфера \(ср\. 8\.7\)

🎯 В вишлисте: 12 книг
```

---

## 15. Что НЕ реализовано в v1 (оставить на v2)

- Экспорт в CSV/Excel
- Годовой читательский челлендж (прогресс-бар)
- Теги к книгам (помимо жанра)
- Персистентные сессии (сессии сбрасываются при рестарте)
- Обложки книг (cover_url сохраняется, но не отображается — Telegram требует inline photo)