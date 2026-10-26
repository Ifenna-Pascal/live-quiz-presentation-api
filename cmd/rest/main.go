package main

import (
	// The blank import runs the init() function of the generated docs package,
	// which registers the API description so that /swagger/doc.json can serve it.
	// Regenerate it after changing any annotation:
	//
	//	swag init -g cmd/rest/main.go
	_ "github.com/ifenna-pascal/interactive-presentation/docs"
	intiator "github.com/ifenna-pascal/interactive-presentation/initiator"
)

// @title			Interactive Presentation API
// @version		2.0
// @description	Run live quiz presentations. A host creates a presentation made of polls and shares its link; players join with a name (and get a secret `player_token`).
// @description	The host starts the presentation, steps through the polls (next / previous), reveals each answer, and finishes. Players vote on the poll on screen.
// @description	Only revealed polls score on the public leaderboard, so it never gives an answer away. Players and the host share a chat.
// @description	Live updates: open `GET /presentations/{presentation_id}/events` (Server-Sent Events) to hear about every change instantly.
// @description	Auth: log in for a 15-minute access token and a single-use refresh token; swap it at `POST /users/token/refresh` before the access token expires.
// @description	Every error response is `{"error": "<message>"}`.
// @BasePath		/
// @schemes		http
//
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Type "Bearer " followed by the access_token returned by POST /users/login or POST /users/token/refresh.
func main() {
	intiator.Init()
}
