run-postgres:
	docker run --name my-postgres -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:latest

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

migrate-db-up:
	migrate -path db/migration/ -database "postgresql://root:secret@localhost:5432/basic_bank_app?sslmode=disable" -verbose up

migrate-db-down:
	migrate -path db/migration/ -database "postgresql://root:secret@localhost:5432/basic_bank_app?sslmode=disable" -verbose down

sqlc:
	sqlc generate

test:
	go test -v -cover ./...
