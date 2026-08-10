package main

import (
	"log"

	"github.com/asnakech/asnakech-servers/internal/config"
	"github.com/asnakech/asnakech-servers/internal/server"
)

// @title           Asnakech School API
// @version         0.1.0
// @description     API for Asnakech School Platform
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.asnakech.com/support
// @contact.email  support@asnakech.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1
func main() {
	cfg := config.Load()

	srv := server.New(cfg)
	if err := srv.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
