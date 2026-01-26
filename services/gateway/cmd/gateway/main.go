package main

import "github.com/artmexbet/raibecas/services/gateway/internal/app"

func main() {
	gateway := app.New()
	if err := gateway.Run(); err != nil {
		panic(err)
	}
}
