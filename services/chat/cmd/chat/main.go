package main

import "github.com/artmexbet/raibecas/services/chat/internal/app"

func main() {
	_app := app.New()
	panic(_app.Run())
}
