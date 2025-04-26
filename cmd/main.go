package main

import (
	log "log/slog"
	"os"

	"github.com/michaelprice232/eks-env-scaledown/config"
	"github.com/michaelprice232/eks-env-scaledown/internal/service"
)

func main() {
	c, err := config.NewConfig()
	if err != nil {
		log.Error("creating config", "error", err)
		os.Exit(1)
	}

	s, err := service.NewService(c)
	if err != nil {
		log.Error("creating service", "error", err)
		os.Exit(2)
	}

	if err = s.Run(); err != nil {
		log.Error("running", "error", err)
		os.Exit(3)
	}
}
