.PHONY: build test deploy clean run-local install deps

# Instalar todas as dependências
deps:
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Build para Lambda
build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bootstrap cmd/api/main.go
	zip deployment.zip bootstrap

# Build local
build-local:
	go build -o bin/api cmd/api/main.go

deploy-aws: build
	aws lambda update-function-code --function-name applywise-api --zip-file fileb://deployment.zip
	aws lambda wait function-updated --function-name applywise-api

# Run localmente
run-local:
	go run cmd/api/main.go

# Testes
test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Testes com cache limpo
test-fresh:
	go clean -testcache
	go test -v -race -coverprofile=coverage.out ./...

# Lint
lint:
	golangci-lint run --timeout=5m

# Format
fmt:
	go fmt ./...
	goimports -w .

# Vet
vet:
	go vet ./...

# Terraform commands
terraform-init:
	cd terraform/environments/dev && terraform init

terraform-plan:
	cd terraform/environments/dev && terraform plan

terraform-apply:
	cd terraform/environments/dev && terraform apply

terraform-destroy:
	cd terraform/environments/dev && terraform destroy

# Deploy completo
deploy: build terraform-apply

# Limpar artifacts
clean:
	rm -f bootstrap deployment.zip coverage.out coverage.html
	rm -rf bin/

# Rodar DynamoDB local (para testes)
dynamodb-local:
	docker run -p 8000:8000 amazon/dynamodb-local
