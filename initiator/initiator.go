package intiator

import (
	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/ifenna-pascal/interactive-presentation/internal/glue/routing"
	playerHttpV1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/player/http/v1"
	presentationHttpV1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/presentation/http/v1"
	usersHttpV1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/users/http/v1"
	voteHttpV1 "github.com/ifenna-pascal/interactive-presentation/internal/handler/vote/http/v1"

	"github.com/ifenna-pascal/interactive-presentation/internal/module/player"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/presentation"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/users"
	"github.com/ifenna-pascal/interactive-presentation/internal/module/vote"
	db "github.com/ifenna-pascal/interactive-presentation/internal/storage/persistence"

	"github.com/ifenna-pascal/interactive-presentation/pkg/config"
	"github.com/ifenna-pascal/interactive-presentation/pkg/middleware"
	"github.com/ifenna-pascal/interactive-presentation/pkg/token"
	"github.com/ifenna-pascal/interactive-presentation/platform/routers"
)

func Init() {
	logger := logrus.New()
	config, err := config.LoadConfig(".")
	if err != nil {
		logger.Fatalf("%s cannot load config", err.Error())
	}

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		logger.Fatal("conn error: ", err)
	}
	defer conn.Close()

	m, err := migrate.New(
		config.MigrationURL,
		config.DBSource,
	)
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

	tokenMaker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		logger.Fatal("token maker error: ", err)
	}
	authMiddleware := middleware.AuthMiddleware(tokenMaker)

	presentationUseCase := presentation.Initialize(store)
	presentationHandler := presentationHttpV1.NewPresentationHandler(logger, presentationUseCase)
	presentationRouting := routing.PresentationRouting(presentationHandler, authMiddleware)

	voteUseCase := vote.Initialize(store)
	voteHandler := voteHttpV1.NewVoteHandler(logger, voteUseCase)
	voteRouting := routing.VoteRouting(voteHandler, authMiddleware)

	playerUseCase := player.Initialize(store)
	playerHandler := playerHttpV1.NewPlayerHandler(logger, playerUseCase)
	playerRouting := routing.PlayerRouting(playerHandler, authMiddleware)

	userUseCase := users.Initialize(store, tokenMaker, config.AccessTokenDuration)
	userHandler := usersHttpV1.NewUserHandler(logger, userUseCase)
	userRouting := routing.UserRouting(userHandler)

	var routersList []routers.Route
	routersList = append(routersList, userRouting...)
	routersList = append(routersList, presentationRouting...)
	routersList = append(routersList, voteRouting...)
	routersList = append(routersList, playerRouting...)
	server := routers.NewRouting(config.ServerAddress, routersList, config.AllowedOrigins())
	server.Serve()
}
