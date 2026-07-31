// @title           Jonly API
// @version         1.0
// @description     Jonly — jonli video-dars platformasi backend API.
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Value format: **Bearer &lt;token&gt;**
package main

import (
	"log"

	"github.com/joho/godotenv"

	"github.com/zoom/darsly/internal/app"
	"github.com/zoom/darsly/internal/pkg/config"
)

func main() {
	// .env mavjud bo'lsa yuklaydi; productionda tashqaridan inject qilinadi.
	_ = godotenv.Load()

	cfg := config.Load()
	if err := app.Run(cfg); err != nil {
		log.Fatalf("app run: %v", err)
	}
}
