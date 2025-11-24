package main

import "github.com/artmexbet/raibecas/services/chat/internal/app"

func main() {
	_app := app.New()
	err := _app.Run()
	if err != nil {
		panic(err)
	}
}
