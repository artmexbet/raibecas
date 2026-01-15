package app

import (
	"os"
	"os/signal"
	"syscall"
)

type App struct {
}

func New() *App {
	return &App{}
}

func (a *App) Run() error {
	// Start services
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-quit
	// Shutdown
	return nil
}
