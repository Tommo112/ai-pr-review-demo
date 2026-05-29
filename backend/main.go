package main

import (
	"demo/backend/internal/app"
	"demo/backend/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	if err := app.New(cfg).Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}
