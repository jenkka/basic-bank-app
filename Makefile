.PHONY: cluster-up cluster-down ingress-install grant-ci bootstrap deploy redeploy destroy logs status run-postgres start-postgres stop-postgres rm-postgres create-db drop-db migrateup migrateup1 migratedown migratedown1 sqlc mock test racetest server

cluster-up:
	eksctl create cluster -f eks/eks.yaml

cluster-down:
	eksctl delete cluster -f eks/eks.yaml

ingress-install:
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.11.2/deploy/static/provider/aws/deploy.yaml
	kubectl wait --namespace ingress-nginx \
		--for=condition=ready pod \
		--selector=app.kubernetes.io/component=controller \
		--timeout=180s

grant-ci:
	eksctl create iamidentitymapping --cluster dummy-bank --region us-east-2 \
		--arn arn:aws:iam::417441726608:user/github-ci --group system:masters --username github-ci

bootstrap: cluster-up ingress-install grant-ci deploy

deploy:
	kubectl apply -f eks/deployment.yaml
	kubectl apply -f eks/service.yaml
	kubectl apply -f eks/ingress.yaml

redeploy:
	kubectl rollout restart deployment dummy-bank-api-deployment

destroy:
	kubectl delete -f eks/ingress.yaml
	kubectl delete -f eks/service.yaml
	kubectl delete -f eks/deployment.yaml

status:
	@echo "=== Pods ===" && kubectl get pods
	@echo "\n=== Service ===" && kubectl get svc
	@echo "\n=== Nodes ===" && kubectl get nodes

logs:
	kubectl logs -l app=dummy-bank-api --tail=50 -f

run-postgres:
	docker network inspect bank-network >/dev/null 2>&1 || docker network create bank-network
	docker run --name dummy-bank-postgres --network bank-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:17-alpine

start-postgres:
	docker start dummy-bank-postgres

stop-postgres:
	docker stop dummy-bank-postgres

rm-postgres:
	docker rm dummy-bank-postgres

create-db:
	docker exec -it dummy-bank-postgres createdb --username=root --owner=root dummy_bank

drop-db:
	docker exec -it dummy-bank-postgres dropdb dummy_bank

migrateup:
	migrate -path db/migration/ -database "postgresql://root:secret@localhost:5432/dummy_bank?sslmode=disable" -verbose up

migrateup1:
	migrate -path db/migration/ -database "postgresql://root:secret@localhost:5432/dummy_bank?sslmode=disable" -verbose up 1

migratedown:
	migrate -path db/migration/ -database "postgresql://root:secret@localhost:5432/dummy_bank?sslmode=disable" -verbose down

migratedown1:
	migrate -path db/migration/ -database "postgresql://root:secret@localhost:5432/dummy_bank?sslmode=disable" -verbose down 1

sqlc:
	sqlc generate

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/jenkka/dummy-bank/db/sqlc Store

test:
	go test -v -cover -timeout 5m ./...

racetest:
	go test -v -race -cover -timeout 5m ./...

server:
	go run main.go
