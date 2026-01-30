package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"

	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/users/internal/config"
	"github.com/artmexbet/raibecas/services/users/internal/handler"
	"github.com/artmexbet/raibecas/services/users/internal/postgres"
	"github.com/artmexbet/raibecas/services/users/internal/server"
	"github.com/artmexbet/raibecas/services/users/internal/service"
)

type App struct {
	server server.Server
	cfg    config.Config
	pg     *postgres.Postgres
	client *natsw.Client
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	natsConn, err := nats.Connect(cfg.NATS.GetURL())
	if err != nil {
		return nil, err
	}

	pg, err := postgres.New(context.Background(), cfg.Database)
	if err != nil {
		return nil, err
	}

	client := natsw.NewClient(natsConn,
		natsw.WithLogger(slog.Default()),
		natsw.WithRecover(),
	)

	svc := service.New(pg)
	h := handler.New(svc)

	srv := server.New(client, h)

	return &App{
		server: srv,
		cfg:    cfg,
		pg:     pg,
		client: client,
	}, nil
}

func (a *App) Run() error {
	err := a.server.Start()
	if err != nil {
		return err
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	a.pg.Close()
	a.client.Close()
	return nil
}
