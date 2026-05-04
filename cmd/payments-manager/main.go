package main

import (
	"log/slog"
	"os"

	"github.com/sunriseex/finance-manager/internal/commands"
	"github.com/sunriseex/finance-manager/internal/config"
	"github.com/sunriseex/finance-manager/pkg/errors"
)

func main() {
	if err := config.Init(); err != nil {
		slog.Error("Ошибка инициализации конфигурации", "error", err)
		os.Exit(1)
	}

	if err := commands.Execute(); err != nil {
		userMsg := errors.GetUserFriendlyMessage(err)
		slog.Error("Ошибка выполнения команды",
			"command", os.Args[1],
			"error", userMsg,
			"details", err)
		os.Exit(1)
	}
}
