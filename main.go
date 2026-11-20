package main

import (
	"log"

	"app/internal/config"
	"app/internal/database"
	"app/internal/server"
)

func main() {
	cfg := config.Load()
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := database.Migrate(db, cfg); err != nil {
		log.Fatal(err)
	}
	log.Fatal(server.New(db, cfg).Run(":" + cfg.Port))
}
