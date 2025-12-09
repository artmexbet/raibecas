package main

import "github.com/artmexbet/raibecas/services/index/internal/app"

func main() {
	a := app.New()
	if err := a.Run(); err != nil {
		panic(err)
	}
}
