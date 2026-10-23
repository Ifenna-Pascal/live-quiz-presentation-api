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
// @version		1.0
// @description	Create presentations made of polls, step through the polls one at a time, and collect votes on the poll currently displayed.
// @BasePath		/
// @schemes		http
//
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Type "Bearer " followed by the token returned by POST /users/login.
func main() {
	intiator.Init()
}
