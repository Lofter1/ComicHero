package main

import (
	"log"

	"github.com/Lofter1/ComicHero/backend/internal/app"
	"github.com/Lofter1/ComicHero/backend/internal/config"
)

var version = "dev"

func main() {
	if err := config.LoadEnvFiles(".env", "../.env"); err != nil {
		log.Printf("failed to load environment files: %v", err)
	}

	if err := app.Run(config.FromEnv(version)); err != nil {
		log.Fatal(err)
	}
}
