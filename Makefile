run-postgres:
	docker network inspect bank-network >/dev/null 2>&1 || docker network create bank-network
	docker run --name my-postgres --network bank-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:17-alpine

start-postgres:
	docker start my-postgres

stop-postgres:
	docker stop my-postgres

rm-postgres:
	docker rm my-postgres

create-db:
	docker exec -it my-postgres createdb --username=root --owner=root dummy_bank

drop-db:
	docker exec -it my-postgres dropdb dummy_bank

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
	mockgen -package mockdb -destination db/mock/store.go github.com/jenkka/dummy-app/db/sqlc Store

test:
	go test -v -race -cover -timeout 5m ./...

server:
	go run main.go
