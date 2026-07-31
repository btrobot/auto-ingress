.PHONY: build test clean help push build-all

# Variables
IMAGE_NAME ?= auto-ingress
IMAGE_TAG ?= dev
DNS_PLUGINS ?= tencentcloud
REGISTRY ?= ccr.ccs.tencentyun.com/nodecoda

# Colors
GREEN = \033[0;32m
YELLOW = \033[0;33m
RED = \033[0;31m
NC = \033[0m

help: ## Show this help
	@echo "$(GREEN)Auto Ingress - Build Commands$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'

build: ## Build the Docker image
	@echo "$(GREEN)Building with DNS plugins: $(DNS_PLUGINS)$(NC)"
	docker build \
		--build-arg DNS_PLUGINS="$(DNS_PLUGINS)" \
		--build-arg VERSION=$(IMAGE_TAG) \
		--build-arg BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		--build-arg VCS_REVISION=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown") \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		.

build-tencentcloud: ## Build with Tencent Cloud DNS
	$(MAKE) build DNS_PLUGINS=tencentcloud

build-aliyun: ## Build with Alibaba Cloud DNS
	$(MAKE) build DNS_PLUGINS=aliyun

build-cloudflare: ## Build with Cloudflare DNS
	$(MAKE) build DNS_PLUGINS=cloudflare

build-route53: ## Build with AWS Route53 DNS
	$(MAKE) build DNS_PLUGINS=route53

build-multi: ## Build with multiple DNS plugins
	$(MAKE) build DNS_PLUGINS="tencentcloud cloudflare route53"

build-all: ## Build all plugin variants
	$(MAKE) build-tencentcloud IMAGE_TAG=$(IMAGE_TAG)-tencentcloud
	$(MAKE) build-aliyun IMAGE_TAG=$(IMAGE_TAG)-aliyun
	$(MAKE) build-cloudflare IMAGE_TAG=$(IMAGE_TAG)-cloudflare
	$(MAKE) build-route53 IMAGE_TAG=$(IMAGE_TAG)-route53

test: ## Run configuration tests
	@echo "$(GREEN)Testing configurations...$(NC)"
	./bin/test-config examples/generic.env
	./bin/test-config examples/nodecoda.env
	./bin/test-config examples/multi-site.env
	@echo "$(GREEN)All tests passed!$(NC)"

generate: ## Generate Caddyfile from example config
	./bin/generate-config examples/generic.env Caddyfile.example
	@echo "$(GREEN)Generated Caddyfile.example$(NC)"

push: ## Push image to registry
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
	docker push $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

clean: ## Clean generated files
	rm -f Caddyfile.example Caddyfile.test
	rm -rf .docker/

lint: ## Lint Dockerfile
	hadolint Dockerfile || echo "$(YELLOW)Install hadolint for Dockerfile linting$(NC)"

scan: ## Scan image for vulnerabilities
	trivy image $(IMAGE_NAME):$(IMAGE_TAG) || echo "$(YELLOW)Install trivy for vulnerability scanning$(NC)"

all: build test generate ## Build, test, and generate

.DEFAULT_GOAL := help
