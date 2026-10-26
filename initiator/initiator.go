package intiator

import (
	"database/sql"
	"net/http"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/ifenna-pascal/interactive-presentation/internal/glue/routing"
	chatHttpV1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/chat/http/v1"
	"github.com/ifenna-pascal/interactive-presentation/internal/handler/health"
	playerHttpV1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/player/http/v1"
	presentationHttpV1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/presentation/http/v1"
	realtimeHttpV1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/realtime/http/v1"
	usersHttpV1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/users/http/v1"
	voteHttpV1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/vote/http/v1"

	"github.com/ifenna-pascal/interactive-presentation/internal/module/chat"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/player"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/presentation"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/realtime"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/users"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/vote"
	db "github.com/ifenna-pascal/interactive-presentation/internal/storage/persistence"

	"github.com/ifenna-pascal/interactive-presentation/pkg/config"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/ifenna-pascal/interactive-presentation/pkg/token"
	"github.com/ifenna-pascal/interactive-presentation/platform/events"
	"github.com/ifenna-pascal/interactive-presentation/platform/routers"
)

func Init() {
	logger := logrus.New()
	config, err := config.LoadConfig(".")
	if err != nil {
		logger.Fatalf("cannot load config: %s", err.Error())
	}
	if !config.IsProduction() && len(config.AllowedOrigins()) == 0 {
		logger.Warn("CORS_ALLOWED_ORIGINS is empty: any website may call the API (fine for development only)")
	}

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		logger.Fatal("conn error: ", err)
	}
	defer conn.Close()

	m, err := migrate.New(config.MigrationURL, config.DBSource)
	if err != nil {
		logger.Fatal("migration error: ", err)
	}
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			logger.Println("No migrations to apply.")
		} else {
			logger.Fatal(err)
		}
	}
	logger.Println("Migrations applied successfully.")

	store := db.NewStore(conn)

	// Real-time: writes NOTIFY, every instance LISTENs and fans out over SSE.
	hub := events.NewHub()
	if err := events.Listen(config.DBSource, hub, logger); err != nil {
		logger.Fatal("events listener: ", err)
	}
	publisher := events.NewPostgresPublisher(conn, logger)

	tokenMaker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		logger.Fatal("token maker error: ", err)
	}
	auth := middleware.AuthMiddleware(tokenMaker)
	optionalAuth := middleware.OptionalAuthMiddleware(tokenMaker)

	presentationHandler := presentationHttpV1.NewPresentationHandler(logger, presentation.Initialize(store, publisher))
	voteHandler := voteHttpV1.NewVoteHandler(logger, vote.Initialize(store, publisher))
	playerHandler := playerHttpV1.NewPlayerHandler(logger, player.Initialize(store, publisher))
	userHandler := usersHttpV1.NewUserHandler(logger, users.Initialize(store, tokenMaker, config.AccessTokenDuration, config.RefreshTokenDuration))
	chatHandler := chatHttpV1.NewChatHandler(logger, chat.Initialize(store, publisher))
	realtimeHandler := realtimeHttpV1.NewRealtimeHandler(logger, realtime.Initialize(store, hub))

	var routersList []routers.Route
	routersList = append(routersList, routers.Route{Method: http.MethodGet, Path: "/ready", Handle: health.ReadyHandler(store)})
	routersList = append(routersList, routing.UserRouting(userHandler)...)
	routersList = append(routersList, routing.PresentationRouting(presentationHandler, auth, optionalAuth)...)
	routersList = append(routersList, routing.VoteRouting(voteHandler, auth)...)
	routersList = append(routersList, routing.PlayerRouting(playerHandler, auth)...)
	routersList = append(routersList, routing.ChatRouting(chatHandler, optionalAuth)...)
	routersList = append(routersList, routing.RealtimeRouting(realtimeHandler)...)

	server := routers.NewRouting(config.ServerAddress, routersList, config.AllowedOrigins())
	server.Serve()
}
