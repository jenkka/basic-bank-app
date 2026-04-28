run-postgres:
	docker network inspect bank-network >/dev/null 2>&1 || docker network create bank-network
	docker run --name my-postgres --network bank-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:latest

start-postgres:
	docker start my-postgres

stop-postgres:
	docker stop my-postgres

rm-postgres:
	docker rm my-postgres

create-db:
	docker exec -it my-postgres createdb --username=root --owner=root basic_bank_app

drop-db:
	docker exec -it my-postgres dropdb basic_bank_app

migrateup:
	migrate -path db/migration/ -database "postgresql://root:secret@localhost:5432/basic_bank_app?sslmode=disable" -verbose up

migrateup1:
	migrate -path db/migration/ -database "postgresql://root:secret@localhost:5432/basic_bank_app?sslmode=disable" -verbose up 1

migratedown:
	migrate -path db/migration/ -database "postgresql://root:secret@localhost:5432/basic_bank_app?sslmode=disable" -verbose down

migratedown1:
	migrate -path db/migration/ -database "postgresql://root:secret@localhost:5432/basic_bank_app?sslmode=disable" -verbose down 1

sqlc:
	sqlc generate

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/jenkka/basic-bank-app/db/sqlc Store

test:
	go test -v -race -cover -timeout 5m ./...

server:
	go run main.go
