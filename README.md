# LivePoll API

A Go REST API for running **live, quiz-style presentations**. A presenter builds a presentation out of polls, players join with a name, and the presenter steps through the polls one at a time while players vote from their own devices. Polls can have correct answers, so the presenter also gets a live leaderboard.

Built with **Gin**, **PostgreSQL**, **sqlc**, **golang-migrate**, and **JWT** auth, with interactive **Swagger** docs.

---

## Features

- **Accounts** – presenters sign up and log in to get a JWT access token.
- **Presentations made of polls** – each poll has a question and lettered options; mark one or more options as correct to score it, or leave none for an opinion poll.
- **Player lobby** – players register by name before the presentation starts. Once the presenter starts it, registration closes.
- **Step-through flow** – only the poll currently on screen accepts votes. The presenter moves forward one poll at a time.
- **One vote per player per poll** – enforced by the database.
- **Answers stay hidden** – correct answers are never returned to players.
- **Leaderboard** – players ranked by correct answers (ties share a rank). Owner-only, so players can't use score changes to work out answers.
- **Vote history** – every vote in a presentation, with correctness, for the owner.
- **Auto migrations** on startup, **CORS** support, and **live reload** with `air` for development.

---

## How a session works

```
Presenter                                   Players
─────────                                   ───────
1. POST /users, POST /users/login
2. POST /presentations   (polls + answers)
3. Share the presentation ID  ───────────▶  4. POST /presentations/{id}/players
5. POST /presentations/{id}/start           (registration now closed)
6. PUT  /presentations/{id}/polls/current   7. GET  /presentations/{id}/polls/current
   (show the next poll)                      8. POST /presentations/{id}/polls/current/votes
9. GET  /presentations/{id}/leaderboard
   …repeat 6–9 for each poll
```

---

## Tech stack

| Area            | Tool                                                                 |
| --------------- | -------------------------------------------------------------------- |
| Language        | Go 1.26                                                              |
| HTTP framework  | [Gin](https://github.com/gin-gonic/gin)                              |
| Database        | PostgreSQL 14                                                        |
| Queries         | [sqlc](https://sqlc.dev) (type-safe Go generated from SQL)           |
| Migrations      | [golang-migrate](https://github.com/golang-migrate/migrate)          |
| Auth            | JWT ([golang-jwt](https://github.com/golang-jwt/jwt)) + bcrypt       |
| Config          | [Viper](https://github.com/spf13/viper)                              |
| API docs        | [swaggo/swag](https://github.com/swaggo/swag) + gin-swagger          |
| Dev tooling     | Docker Compose, [air](https://github.com/air-verse/air) live reload  |

---

## Getting started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- Go 1.26+ (only if you want to run the API outside Docker)

### Run everything with Docker

```bash
git clone https://github.com/Ifenna-Pascal/<repo-name>.git
cd <repo-name>
cp app.env.example app.env   # then edit the values
docker compose up --build
```

The API starts on **http://localhost:8081**. Migrations run automatically on startup.

- Health check: http://localhost:8081/ping
- Swagger UI: http://localhost:8081/swagger/index.html

### Run locally with live reload

Start only the database in Docker, then run the API on your machine with `air`:

```bash
docker compose up -d postgres

go install github.com/air-verse/air@latest
go install github.com/swaggo/swag/cmd/swag@latest

air
```

`air` regenerates the Swagger docs, rebuilds, and restarts the server on every change to a `.go`, `.sql`, or `.env` file. It also points `DB_SOURCE` at `localhost` instead of the `postgres` container name.

---

## Configuration

Settings are read from `app.env` and can be overridden with environment variables of the same name. `app.env` is git-ignored; copy [app.env.example](app.env.example) to create it.

| Variable                 | Description                                                         | Default (dev)                      |
| ------------------------ | ------------------------------------------------------------------- | ---------------------------------- |
| `SERVER_ADDRESS`         | Address the API listens on                                          | `0.0.0.0:8081`                     |
| `ENVIRONMENT`            | Environment name                                                    | `development`                      |
| `DB_DRIVER`              | Database driver                                                     | `postgres`                         |
| `DB_SOURCE`              | PostgreSQL connection string                                        | `postgresql://root:secret@postgres:5432/...` |
| `MIGRATION_URL`          | Location of migration files                                         | `file://db/migration`              |
| `TOKEN_SYMMETRIC_KEY`    | Secret used to sign JWTs                                            | dev-only value                     |
| `ACCESS_TOKEN_DURATION`  | How long an access token is valid                                   | `15m`                              |
| `CORS_ALLOWED_ORIGINS`   | Comma-separated websites allowed to call the API (empty = any)      | empty                              |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | Used by the Postgres container        | `root` / `secret` / `interactive_presenations` |

> ⚠️ Change `TOKEN_SYMMETRIC_KEY` and the database password before deploying anywhere public.

---

## API reference

Full request and response schemas are in Swagger at `/swagger/index.html`. 🔒 = needs `Authorization: Bearer <token>` from `POST /users/login`.

### Users

| Method | Path           | Description              |
| ------ | -------------- | ------------------------ |
| POST   | `/users`       | Create an account        |
| POST   | `/users/login` | Log in and get a token   |

### Presentations

| Method | Path                                         | Auth | Description                                       |
| ------ | -------------------------------------------- | ---- | ------------------------------------------------- |
| POST   | `/presentations`                             | 🔒   | Create a presentation with its polls              |
| GET    | `/presentations/{presentation_id}`           |      | Get a presentation and its polls                  |
| GET    | `/users/me/presentations?limit=&offset=`     | 🔒   | List your presentations, newest first             |
| GET    | `/presentations/{presentation_id}/polls/current` |  | Get the poll currently on screen                  |
| PUT    | `/presentations/{presentation_id}/polls/current` | 🔒 | Move to the next poll                           |

### Players

| Method | Path                                       | Auth | Description                                   |
| ------ | ------------------------------------------ | ---- | --------------------------------------------- |
| POST   | `/presentations/{presentation_id}/players` |      | Join a presentation (before it starts)        |
| GET    | `/presentations/{presentation_id}/players` |      | List registered players                       |
| POST   | `/presentations/{presentation_id}/start`   | 🔒   | Start the presentation and close registration |

### Votes and scoring

| Method | Path                                                     | Auth | Description                                |
| ------ | -------------------------------------------------------- | ---- | ------------------------------------------ |
| POST   | `/presentations/{presentation_id}/polls/current/votes`   |      | Vote on the current poll                   |
| GET    | `/presentations/{presentation_id}/polls/{poll_id}/votes` |      | Vote counts for any poll                   |
| GET    | `/presentations/{presentation_id}/votes`                 | 🔒   | Every vote, with correctness (owner only)  |
| GET    | `/presentations/{presentation_id}/leaderboard`           | 🔒   | Player rankings (owner only)               |

### Example

```bash
# Sign up and log in
curl -X POST localhost:8081/users -H 'Content-Type: application/json' \
  -d '{"name":"Ada Lovelace","email":"ada@example.com","password":"s3cretpass"}'

TOKEN=$(curl -s -X POST localhost:8081/users/login -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"s3cretpass"}' | jq -r .token)

# Create a presentation
curl -X POST localhost:8081/presentations \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{
        "polls": [
          {
            "question": "Which language is this API written in?",
            "options": [
              {"key": "A", "value": "Go", "correct": true},
              {"key": "B", "value": "Rust"}
            ]
          }
        ]
      }'

# A player joins
curl -X POST localhost:8081/presentations/<presentation_id>/players \
  -H 'Content-Type: application/json' -d '{"player_name":"Grace"}'

# Start, show the first poll, then vote
curl -X POST localhost:8081/presentations/<presentation_id>/start -H "Authorization: Bearer $TOKEN"
curl -X PUT  localhost:8081/presentations/<presentation_id>/polls/current -H "Authorization: Bearer $TOKEN"
curl -X POST localhost:8081/presentations/<presentation_id>/polls/current/votes \
  -H 'Content-Type: application/json' \
  -d '{"poll_id":"<poll_id>","key":"A","player_id":"<player_id>"}'
```

---

## Project structure

```
.
├── cmd/rest/            # Entry point (main.go) and Swagger annotations
├── initiator/           # Wires config, DB, migrations, use cases, handlers, routes
├── internal/
│   ├── constant/
│   │   ├── model/       # Request/response DTOs and domain errors
│   │   └── query/       # Raw SQL queries consumed by sqlc
│   ├── glue/routing/    # Route definitions per domain
│   ├── handler/         # HTTP handlers (presentation, player, vote, users)
│   ├── module/          # Business logic (use cases)
│   └── storage/persistence/  # sqlc-generated code + transactions
├── pkg/
│   ├── config/          # Viper config loader
│   ├── middleware/      # Auth (JWT) and content-type middleware
│   ├── token/           # JWT maker
│   └── util/            # Password hashing
├── platform/routers/    # Gin server, CORS, health check, Swagger route
├── db/migration/        # SQL migrations (golang-migrate)
├── docs/                # Generated Swagger docs (do not edit by hand)
├── Dockerfile
├── docker-compose.yml
└── .air.toml
```

---

## Development

**Regenerate database code** after editing files in `db/migration/` or `internal/constant/query/`:

```bash
sqlc generate
```

**Add a migration:**

```bash
migrate create -ext sql -dir db/migration -seq <name>
```

**Regenerate Swagger docs** after changing handler annotations (`air` does this for you):

```bash
swag init -g cmd/rest/main.go
```

---

## Contributing

1. Fork the repo and create a branch: `git checkout -b feat/my-change`
2. Make your changes and run the API locally to check them
3. Commit and open a pull request

---

## Author

**Ifenna Pascal** – [@Ifenna-Pascal](https://github.com/Ifenna-Pascal)
