// @title           SPSync API
// @version         1.0
// @description     HTTP API server for the SPSync platform.
// @host            localhost:8080
// @BasePath        /api/v1
//
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     JWT access token. Example: Bearer {token}
package main

import (
	"os"

	"spsyncapi/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
