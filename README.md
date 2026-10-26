# LivePoll API

A Go REST API for running **live, quiz-style presentations**. A presenter builds a presentation out of polls, players join with a name, and the presenter steps through the polls one at a time while players vote from their own devices. Polls can have correct answers, so everyone also gets a live leaderboard. Updates stream to every screen instantly, and players and the presenter can chat.

The React frontend lives in [`../interactive-presentation-web`](../interactive-presentation-web).

Built with **Gin**, **PostgreSQL**, **sqlc**, **golang-migrate**, and **JWT** auth, with interactive **Swagger** docs.

---

## Features

- **Accounts** – presenters sign up and log in to get a short-lived JWT access token and a single-use refresh token. Refreshing rotates the refresh token; reusing an old one ends every session of that user.
- **Presentations made of polls** – each poll has a question and lettered options; mark one or more options as correct to score it, or leave none for an opinion poll.
- **Player lobby** – players register by name before the presentation starts and get a secret `player_token` that they must send to vote and chat. Once the presenter starts, registration closes. The presenter can remove players.
- **Timed questions, fastest wins** – each poll has its own time limit (5–600 s, set when creating). The countdown starts when the poll is first shown. The first player to pick a correct option wins the poll: it closes on the spot and they get the point. A wrong pick can be followed by another option (each option once) until time runs out; opinion polls take one answer each. Answers to a poll are processed one at a time under a row lock, so exactly one player can win even when several answer at the same instant.
- **Step-through flow** – only the poll currently on screen accepts votes. The presenter moves forward or back, reveals the answer (which closes voting on that poll), and finishes the presentation.
- **Edit and delete** – presentations (with a title) can be edited until they start, and deleted at any time.
- **One vote per player per poll** – enforced by the database.
- **Answers stay hidden** – players only see a poll's correct answer after the presenter reveals it.
- **Leaderboard** – public; one point per poll won (ties share a rank; faster total winning time orders a tie).
- **Chat** – players and the presenter share one chat per presentation.
- **Live updates** – a Server-Sent Events stream per presentation. Writes `pg_notify` and every API instance `LISTEN`s, so it works with several instances.
- **Vote history** – every vote in a presentation, with correctness, for the owner.
- **Auto migrations** on startup, **CORS** support, per-IP **rate limits**, a `/ready` check, and **live reload** with `air` for development.

---

## How a session works

```
Presenter                                   Players
─────────                                   ───────
1. POST /users, POST /users/login
2. POST /presentations   (polls + answers)
3. Share the join link  ─────────────────▶  4. POST /presentations/{id}/players  (→ player_token)
                                             GET  /presentations/{id}/events     (live stream)
5. POST /presentations/{id}/start           (registration closed, poll 1 on screen)
                                             6. GET  /presentations/{id}/polls/current
                                             7. POST /presentations/{id}/polls/current/votes
                                             (first correct answer wins and closes the poll;
                                              stream sends `winner`)
8. POST /presentations/{id}/polls/current/reveal   (optional: end the poll early)
                                             9. GET  /presentations/{id}/leaderboard
10. PUT /presentations/{id}/polls/current   (next poll) …repeat 6–10
11. POST /presentations/{id}/finish
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

The API starts on **http://localhost:8081** and the frontend on **http://localhost:3000** (from `../interactive-presentation-web`). Migrations run automatically on startup.

- Health check: http://localhost:8081/ping · Readiness (checks the database): http://localhost:8081/ready
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
| `REFRESH_TOKEN_DURATION` | How long a refresh token is valid (each refresh issues a new one)   | `168h`                             |
| `CORS_ALLOWED_ORIGINS`   | Comma-separated websites allowed to call the API (empty = any; required in production) | empty           |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | Used by the Postgres container        | `root` / `secret` / `interactive_presenations` |

> ⚠️ Change `TOKEN_SYMMETRIC_KEY` and the database password before deploying anywhere public. With `ENVIRONMENT=production` the API refuses to start with the development key or an empty `CORS_ALLOWED_ORIGINS`.

---

## API reference

Full request and response schemas are in Swagger at `/swagger/index.html`. 🔒 = needs `Authorization: Bearer <token>` from `POST /users/login`.

### Users

| Method | Path           | Description              |
| ------ | -------------- | ------------------------ |
| POST   | `/users`       | Create an account        |
| POST   | `/users/login` | Log in: access token + refresh token |
| POST   | `/users/token/refresh` | Swap a refresh token for a new pair |
| POST   | `/users/logout` | Revoke a refresh token  |

### Presentations

| Method | Path                                         | Auth | Description                                       |
| ------ | -------------------------------------------- | ---- | ------------------------------------------------- |
| POST   | `/presentations`                             | 🔒   | Create a presentation (title + polls)             |
| GET    | `/presentations/{presentation_id}`           |      | Get a presentation and its polls (with answers for the owner) |
| PUT    | `/presentations/{presentation_id}`           | 🔒   | Replace title and polls (before it starts)        |
| DELETE | `/presentations/{presentation_id}`           | 🔒   | Delete it with all players, votes and chat        |
| GET    | `/users/me/presentations?limit=&offset=`     | 🔒   | List your presentations, newest first             |
| GET    | `/presentations/{presentation_id}/polls/current` |  | Get the poll currently on screen                  |
| PUT    | `/presentations/{presentation_id}/polls/current` | 🔒 | Move to the next poll                           |
| PUT    | `/presentations/{presentation_id}/polls/previous` | 🔒 | Move back one poll                             |
| POST   | `/presentations/{presentation_id}/polls/current/reveal` | 🔒 | Reveal the answer and close voting      |
| POST   | `/presentations/{presentation_id}/finish`    | 🔒   | Finish the presentation                           |

### Players

| Method | Path                                       | Auth | Description                                   |
| ------ | ------------------------------------------ | ---- | --------------------------------------------- |
| POST   | `/presentations/{presentation_id}/players` |      | Join a presentation (before it starts)        |
| GET    | `/presentations/{presentation_id}/players` |      | List registered players                       |
| POST   | `/presentations/{presentation_id}/start`   | 🔒   | Start the presentation and close registration |
| DELETE | `/presentations/{presentation_id}/players/{player_id}` | 🔒 | Remove a player                     |

### Votes and scoring

| Method | Path                                                     | Auth | Description                                |
| ------ | -------------------------------------------------------- | ---- | ------------------------------------------ |
| POST   | `/presentations/{presentation_id}/polls/current/votes`   |      | Answer the current poll (needs `player_token`); returns `{correct, won, scored}` |
| GET    | `/presentations/{presentation_id}/polls/{poll_id}/votes` |      | Vote counts for any poll                   |
| GET    | `/presentations/{presentation_id}/votes`                 | 🔒   | Every vote, with correctness (owner only)  |
| GET    | `/presentations/{presentation_id}/leaderboard`           |      | Player rankings (one point per poll won)   |

### Chat and live updates

| Method | Path                                            | Auth | Description                                         |
| ------ | ----------------------------------------------- | ---- | --------------------------------------------------- |
| GET    | `/presentations/{presentation_id}/messages?after=&limit=` | | Chat messages, oldest first                    |
| POST   | `/presentations/{presentation_id}/messages`     | 🔒 or player token | Send a message (host via token, player via `player_id` + `player_token`) |
| GET    | `/presentations/{presentation_id}/events`       |      | Server-Sent Events: `presentation`, `players`, `vote`, `winner`, `chat`, `deleted`, `resync` |

### Example

```bash
# Sign up and log in
curl -X POST localhost:8081/users -H 'Content-Type: application/json' \
  -d '{"name":"Ada Lovelace","email":"ada@example.com","password":"s3cretpass"}'

TOKEN=$(curl -s -X POST localhost:8081/users/login -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"s3cretpass"}' | jq -r .access_token)

# Create a presentation
curl -X POST localhost:8081/presentations \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{
        "title": "Friday quiz",
        "polls": [
          {
            "question": "Which language is this API written in?",
            "time_limit_seconds": 15,
            "options": [
              {"key": "A", "value": "Go", "correct": true},
              {"key": "B", "value": "Rust"}
            ]
          }
        ]
      }'

# A player joins (keep the player_token from the response)
curl -X POST localhost:8081/presentations/<presentation_id>/players \
  -H 'Content-Type: application/json' -d '{"player_name":"Grace"}'

# Start (poll 1 goes on screen), then the player votes
curl -X POST localhost:8081/presentations/<presentation_id>/start -H "Authorization: Bearer $TOKEN"
curl -X POST localhost:8081/presentations/<presentation_id>/polls/current/votes \
  -H 'Content-Type: application/json' \
  -d '{"poll_id":"<poll_id>","key":"A","player_id":"<player_id>","player_token":"<player_token>"}'

# A correct answer wins and closes the poll ({"correct":true,"won":true,...}); check the leaderboard
curl localhost:8081/presentations/<presentation_id>/leaderboard
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
│   ├── handler/         # HTTP handlers (presentation, player, vote, users, chat, realtime, health)
│   │   └── respond/     # One place that turns errors into {"error": ...} responses
│   ├── integration/     # End-to-end use-case test against a real database
│   ├── module/          # Business logic (use cases)
│   └── storage/persistence/  # sqlc-generated code + transactions
├── pkg/
│   ├── config/          # Viper config loader
│   ├── middleware/      # Auth (required/optional JWT), content-type and rate-limit middleware
│   ├── token/           # JWT maker
│   └── util/            # Password hashing and secret tokens
├── platform/
│   ├── events/          # Live updates: Postgres NOTIFY/LISTEN + in-process hub
│   └── routers/         # Gin server, CORS, health check, Swagger route
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

**Run the tests:**

```bash
go test ./...

# Integration test against a throwaway database (it is wiped):
docker exec interactive-presentation-postgres-1 createdb -U root ip_test
TEST_DB_SOURCE="postgresql://root:secret@localhost:5432/ip_test?sslmode=disable" go test ./internal/integration/
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
