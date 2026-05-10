package main

import (
	"log/slog"
	"os"

	"github.com/sunriseex/capitalflow/internal/commands"
	"github.com/sunriseex/capitalflow/internal/config"
	"github.com/sunriseex/capitalflow/pkg/errors"
)

func main() {
	if err := config.Init(); err != nil {
		slog.Error("Ошибка инициализации конфигурации", "error", err)
		os.Exit(1)
	}

	if err := commands.Execute(); err != nil {
		userMsg := errors.GetUserFriendlyMessage(err)
		slog.Error("Ошибка выполнения команды",
			"command", commandName(os.Args),
			"error", userMsg,
			"details", err)
		os.Exit(1)
	}
}

func commandName(args []string) string {
	if len(args) < 2 {
		return "default"
	}
	return args[1]
}
