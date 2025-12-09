lint:
	golangci-lint.exe run --fix .\services\index\... .\services\chat\... .\services\auth\...

setup:
	ollama pull embeddinggemma:300m
	ollama pull gemma3:4b

up:
	docker compose -f ./deploy/docker-compose.dev.yml up -d --build

up-env:
	docker compose -f ./deploy/docker-compose.dev.yml up -d --build postgres redis nats qdrant