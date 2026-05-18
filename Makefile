# ---- Configuration (override on the command line, e.g. `make migrateup DB_URL=...`) ----
AWS_REGION             ?= us-east-2
CLUSTER                ?= dummy-bank
GITHUB_CI_ARN          ?= arn:aws:iam::417441726608:user/github-ci
DB_URL                 ?= postgresql://root:secret@localhost:5432/dummy_bank?sslmode=disable
MIGRATION_PATH         ?= db/migration/
INGRESS_NGINX_MANIFEST ?= https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.11.2/deploy/static/provider/aws/deploy.yaml
CERT_MANAGER_MANIFEST  ?= https://github.com/cert-manager/cert-manager/releases/download/v1.20.2/cert-manager.yaml

.PHONY: help \
	cluster-up cluster-down ingress-install ingress-uninstall \
	cert-manager-install cert-manager-uninstall issuer-install issuer-uninstall \
	grant-ci bootstrap teardown \
	deploy redeploy destroy status logs check-aws \
	run-postgres start-postgres stop-postgres rm-postgres create-db drop-db \
	migrateup migrateup1 migratedown migratedown1 \
	sqlc mock test racetest server

.DEFAULT_GOAL := help

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

##@ EKS lifecycle

cluster-up: ## Create the EKS cluster from eks/eks.yaml
	eksctl create cluster -f eks/eks.yaml

cluster-down: ## Delete the EKS cluster
	eksctl delete cluster -f eks/eks.yaml

ingress-install: ## Install nginx-ingress and wait for it to be ready
	kubectl apply -f $(INGRESS_NGINX_MANIFEST)
	kubectl wait --namespace ingress-nginx \
		--for=condition=ready pod \
		--selector=app.kubernetes.io/component=controller \
		--timeout=180s

ingress-uninstall: ## Remove nginx-ingress (and its cloud load balancer)
	kubectl delete --ignore-not-found=true -f $(INGRESS_NGINX_MANIFEST)

cert-manager-install: ## Install cert-manager (v1.20.2) and wait for it to be ready
	kubectl apply -f $(CERT_MANAGER_MANIFEST)
	kubectl wait --namespace cert-manager \
		--for=condition=ready pod \
		--selector=app.kubernetes.io/instance=cert-manager \
		--timeout=180s

cert-manager-uninstall: ## Remove cert-manager
	kubectl delete --ignore-not-found=true -f $(CERT_MANAGER_MANIFEST)

issuer-install: ## Apply the letsencrypt ClusterIssuer (needs cert-manager first)
	kubectl apply -f eks/issuer.yaml

issuer-uninstall: ## Remove the letsencrypt ClusterIssuer
	kubectl delete --ignore-not-found=true -f eks/issuer.yaml

grant-ci: ## Map the github-ci IAM user into the cluster for CD
	eksctl create iamidentitymapping --cluster $(CLUSTER) --region $(AWS_REGION) \
		--arn $(GITHUB_CI_ARN) --group system:masters --username github-ci

bootstrap: cluster-up ingress-install cert-manager-install issuer-install grant-ci deploy ## Stand up everything from scratch

# Inverse of bootstrap: remove app, issuer, cert-manager, then the ingress
# controller (and its ELB) before deleting the cluster, so no orphaned load
# balancer survives cluster-down.
# RDS is managed manually from the AWS console and intentionally not touched here.
teardown: destroy issuer-uninstall cert-manager-uninstall ingress-uninstall cluster-down ## Tear everything down (inverse of bootstrap)

##@ App deployment

deploy: ## Apply the app deployment, service and ingress
	kubectl apply -f eks/deployment.yaml
	kubectl apply -f eks/service.yaml
	kubectl apply -f eks/ingress.yaml

redeploy: ## Restart the API deployment to pull a fresh image
	kubectl rollout restart deployment dummy-bank-api-deployment

destroy: ## Remove the app deployment, service and ingress
	kubectl delete --ignore-not-found=true -f eks/ingress.yaml
	kubectl delete --ignore-not-found=true -f eks/service.yaml
	kubectl delete --ignore-not-found=true -f eks/deployment.yaml

status: ## Show pods, services and nodes
	@echo "=== Pods ===" && kubectl get pods
	@echo "\n=== Service ===" && kubectl get svc
	@echo "\n=== Nodes ===" && kubectl get nodes

logs: ## Tail the API logs
	kubectl logs -l app=dummy-bank-api --tail=50 -f

check-aws: ## List running EC2 instances and EKS clusters
	@aws ec2 describe-instances --region $(AWS_REGION) \
		--query 'Reservations[].Instances[] | [?State.Name==`running`].[InstanceId,InstanceType,LaunchTime]' \
		--output table
	@aws eks list-clusters --region $(AWS_REGION) --output table

##@ Local database

run-postgres: ## Create and start the local Postgres container
	docker network inspect bank-network >/dev/null 2>&1 || docker network create bank-network
	docker run --name dummy-bank-postgres --network bank-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:17-alpine

start-postgres: ## Start the existing Postgres container
	docker start dummy-bank-postgres

stop-postgres: ## Stop the Postgres container
	docker stop dummy-bank-postgres

rm-postgres: ## Remove the Postgres container
	docker rm dummy-bank-postgres

create-db: ## Create the dummy_bank database
	docker exec -it dummy-bank-postgres createdb --username=root --owner=root dummy_bank

drop-db: ## Drop the dummy_bank database
	docker exec -it dummy-bank-postgres dropdb dummy_bank

migrateup: ## Apply all up migrations
	migrate -path $(MIGRATION_PATH) -database "$(DB_URL)" -verbose up

migrateup1: ## Apply the next up migration
	migrate -path $(MIGRATION_PATH) -database "$(DB_URL)" -verbose up 1

migratedown: ## Roll back all migrations
	migrate -path $(MIGRATION_PATH) -database "$(DB_URL)" -verbose down

migratedown1: ## Roll back the last migration
	migrate -path $(MIGRATION_PATH) -database "$(DB_URL)" -verbose down 1

##@ Codegen & tests

sqlc: ## Regenerate Go bindings from SQL
	sqlc generate

mock: ## Regenerate gomock mocks for the Store interface
	mockgen -package mockdb -destination db/mock/store.go github.com/jenkka/dummy-bank/db/sqlc Store

test: ## Run the full test suite with coverage
	go test -v -cover -timeout 5m ./...

racetest: ## Run tests with the race detector
	go test -v -race -cover -timeout 5m ./...

server: ## Run the API on :8080
	go run main.go
