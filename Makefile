lint:
	golangci-lint.exe run --fix .\services\index\... .\services\chat\... .\services\auth\...

setup:
	ollama pull embeddinggemma:300m
	ollama pull gemma3:4b