# Personal assistant

Personal assistant - локальный Go-сервис, который принимает сообщения, отправляет их в OpenRouter и при необходимости выполняет действия в Jira через tool calls.

[English version](README.md)

## Возможности

- endpoint `POST /chat` для сообщений пользователя
- маршрутизация инструментов из ответа LLM
- выполнение Jira-функций:
  - `create_project`
  - `search_projects`
  - `delete_project`
- хранение истории диалога в памяти процесса
- endpoint `GET /memory` для просмотра текущей истории

## Принцип работы

1. Клиент отправляет JSON в `POST /chat`.
2. Сервис собирает контекст:
   - системный промпт
   - история сообщений
   - новое сообщение пользователя
3. OpenRouter возвращает обычный ответ или `tool_calls`.
4. Если пришел tool call, сервис:
   - парсит аргументы
   - вызывает Jira API
   - отправляет результат инструмента обратно в модель (`role=tool`) и получает финальный ответ
5. Возвращает клиенту JSON-ответ.

## Структура проекта

- `cmd/main.go` - точка входа
- `internal/config` - чтение `settings.json`
- `internal/api` - HTTP-роуты и обработчики
- `internal/ai` - запросы в OpenRouter, tool routing, память
- `internal/jira` - обертка над Jira API
- `internal/logg` - кастомный логгер (`slog` + цветные уровни)
- `internal/models` - DTO, разделенные по доменам:
  - `internal/models/chat`
  - `internal/models/tool`
  - `internal/models/openrouter`
  - `internal/models/jira`

## Конфиг (`settings.json`)

### Основные поля

- `api_key_openrouter` - API ключ OpenRouter. Получить можно здесь: <https://openrouter.ai/settings/keys>
- `model_chat_openrouter` - список чат-моделей. Первая модель основная, остальные используются как fallback.
- `model_embending_openrouter` - embedding-модель для будущей логики long-memory/embeddings.
- `api_url_openrouter` - URL chat completions OpenRouter (обычно `https://openrouter.ai/api/v1/chat/completions`).

### Jira поля

- `jira_api_key` - API токен Atlassian. Получить можно здесь: <https://id.atlassian.com/manage-profile/security/api-tokens>
- `jira_email` - email вашего Atlassian аккаунта.
- `jira_personal_url` - базовый URL Jira в формате `https://<ваш-тег>.atlassian.net`.

### Поля сервиса

- `api_host` - хост, на котором поднимается HTTP API (обычно `localhost`).
- `api_port` - порт HTTP API (обычно `8080`).

### Поля промптов

- `promt_system_chat` - основной системный промпт ассистента.
- `promt_memory_summary` - системный промпт для суммаризации истории.
- `memory_summary_user_promt` - пользовательское сообщение для запуска суммаризации.

### Поля управления контекстом

- `max_tokens_context` - жесткий лимит контекста в токенах.
  - `0` включает авто-режим: лимит рассчитывается по выбранным моделям.
- `high_border_max_context` - запас от верхней границы контекста.
  - пример: при `32000` и запасе `5000` рабочий лимит будет `27000`.
- `summary_memory_step` - порог для шага суммаризации памяти.
- `division_coefficient` - коэффициент деления доступного контекста между зонами памяти.

### Важно по безопасности

`settings.json` хранится в открытом виде. Не коммитьте реальные ключи и токены в репозиторий. Лучше хранить секреты в переменных окружения или локальном файле, который игнорируется git.

## Запуск

### Подготовка MySQL (локальный режим)

Перед первым запуском в режиме local DB создайте БД и пользователя:

```bash
mysql -u root -p < scripts/mysql_bootstrap.sql
```

После этого укажите `local_mysql_dsn` в `settings.json` (или `LOCAL_MYSQL_DSN`) для созданного пользователя, например:
`assistant_app:change_me_strong_password@tcp(127.0.0.1:3306)/assistant?parseTime=true`.

### Старт сервиса

```bash
go run ./cmd
```

Сервис поднимается на `http://<api_host>:<api_port>`.

## API

### `POST /chat`

Тело запроса:

```json
{
  "message": "создай проект TEST в Jira"
}
```

Пример:

```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d "{\"message\":\"покажи проекты Jira\"}"
```

Ответы:

- `200` - JSON от OpenRouter
- `400` - невалидный запрос
- `500` - ошибка обработки (AI/Jira/tool pipeline)

### `GET /memory`

Возвращает текущую историю диалога из памяти.

```bash
curl http://localhost:8080/memory
```

## Логи

Используются уровни:

- `QUESTION` - входящий вопрос
- `TASK` - промежуточные шаги инструментов
- `ANSWER` - финальный ответ модели
- `ERROR` - ошибки обработки и интеграций
- `INFO` - служебные логи сервиса

## Ограничения

- память только в RAM процесса
- нет персистентного хранения истории
- нет встроенной авторизации HTTP-эндпоинтов
- long-memory/embeddings пока не доведены до конца
