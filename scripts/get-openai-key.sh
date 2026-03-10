#!/bin/bash
echo "🤖 Getting OpenAI API Key..."
echo ""
echo "1. Go to: https://platform.openai.com/api-keys"
echo "2. Click 'Create new secret key'"
echo "3. Copy the key (starts with sk-proj-...)"
echo ""
echo "Add this to your secrets.tfvars:"
echo "openai_api_key = \"sk-proj-...\""

# Makefile additions for Terraform
.PHONY: tf-init tf-plan tf-apply tf-destroy tf-output

# Terraform shortcuts
tf-init:
	cd terraform/environments/dev && terraform init

tf-plan:
	cd terraform/environments/dev && terraform plan -var-file="secrets.tfvars"

tf-apply:
	cd terraform/environments/dev && terraform apply -var-file="secrets.tfvars"

tf-destroy:
	cd terraform/environments/dev && terraform destroy -var-file="secrets.tfvars"

tf-output:
	cd terraform/environments/dev && terraform output

tf-validate:
	cd terraform/environments/dev && terraform validate

tf-fmt:
	terraform fmt -recursive terraform/

# Full deploy
deploy-dev: build tf-apply
	@echo "✅ Deployed to Dev!"
	@cd terraform/environments/dev && terraform output api_endpoint