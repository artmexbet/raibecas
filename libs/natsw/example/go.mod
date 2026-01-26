module github.com/artmexbet/raibecas/libs/natsw/example

go 1.23.4

require (
	github.com/artmexbet/raibecas/libs/natsw v0.0.0
	github.com/nats-io/nats.go v1.38.0
	go.opentelemetry.io/otel v1.33.0
	go.opentelemetry.io/otel/trace v1.33.0
)

replace github.com/artmexbet/raibecas/libs/natsw => ../
