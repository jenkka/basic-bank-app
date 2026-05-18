.PHONY: cluster-up cluster-down deploy redeploy destroy logs status run-postgres start-postgres stop-postgres rm-postgres create-db drop-db migrateup migrateup1 migratedown migratedown1 sqlc mock test racetest server

cluster-up:
	eksctl create cluster -f eks/eks.yaml

cluster-down:
	eksctl delete cluster -f eks/eks.yaml

deploy:
	kubectl apply -f eks/deployment.yaml
	kubectl apply -f eks/service.yaml

redeploy:
	kubectl rollout restart deployment simple-bank-api-deployment

destroy:
	kubectl delete -f eks/service.yaml
	kubectl delete -f eks/deployment.yaml

status:
	@echo "=== Pods ===" && kubectl get pods
	@echo "\n=== Service ===" && kubectl get svc
	@echo "\n=== Nodes ===" && kubectl get nodes

logs:
	kubectl logs -l app=simple-bank-api --tail=50 -f

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
