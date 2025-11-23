package app

import (
	"fmt"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	"github.com/artmexbet/raibecas/services/chat/internal/neuro"
	qdrantWrapper "github.com/artmexbet/raibecas/services/chat/internal/qdrant-wrapper"
	"github.com/artmexbet/raibecas/services/chat/internal/service"

	"github.com/qdrant/go-client/qdrant"
)

type App struct {
}

func New() *App {
	return &App{}
}

func (a *App) Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("error loading config: %v", err)
	}

	qdrantClient, err := qdrant.NewClient(&qdrant.Config{
		Host: cfg.Qdrant.Host,
		Port: cfg.Qdrant.Port,
	})
	if err != nil {
		return fmt.Errorf("error creating qdrant client: %v", err)
	}
	qdrantWrap := qdrantWrapper.New(cfg.Qdrant, qdrantClient)

	ollama, err := neuro.NewConnector(cfg.Ollama)
	if err != nil {
		return fmt.Errorf("error creating ollama connector: %v", err)
	}

	svc := service.New(qdrantWrap, ollama)
	_ = svc

	return nil
}
