package main

import (
	"os"

	"github.com/awch-D/ForgeX/forgex-cli/commands"
	"github.com/awch-D/ForgeX/forgex-core/logger"
)

func main() {
	logger.Init("info", true)
	defer logger.Sync()

	if err := commands.Execute(); err != nil {
		logger.L().Errorw("ForgeX exited with error", "error", err)
		os.Exit(1)
	}
}
