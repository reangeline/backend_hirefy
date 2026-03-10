#!/bin/bash

echo "Setting up ApplyWise Backend..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    exit 1
fi

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    echo "Error: AWS CLI is not installed"
    exit 1
fi

# Check if Terraform is installed
if ! command -v terraform &> /dev/null; then
    echo "Error: Terraform is not installed"
    exit 1
fi

# Install dependencies
echo "Installing Go dependencies..."
go mod download

# Install tools
echo "Installing development tools..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Create necessary directories
mkdir -p bin
mkdir -p logs

# Copy .env.example to .env if not exists
if [ ! -f .env ]; then
    echo "Creating .env file..."
    cp .env.example .env
    echo "Please edit .env file with your configuration"
fi

echo "Setup complete!"
echo ""
echo "Next steps:"
echo "1. Edit .env file with your AWS credentials and API keys"
echo "2. Run 'make terraform-init' to initialize Terraform"
echo "3. Run 'make run-local' to start the server locally"
echo "4. Run 'make deploy' to deploy to AWS"
