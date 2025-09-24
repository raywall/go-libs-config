.PHONY: build delete run test clean

build:
	@cd examples && terraform init && terraform plan -out plan.out && terraform apply -auto-approve

delete:
	@cd examples && terraform destroy -auto-approve

run:
	@go run examples/main.go

test:
	@go test ./...

clean: delete
	@cd examples && rm -rf .terraform .terraform.lock.hcl terraform.tfstate* plan.out