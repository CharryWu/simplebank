postgres:
	sudo docker run --name backend-master -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:12-alpine

startdb:
	sudo docker start backend-master

createdb:
	sudo docker exec -it backend-master createdb --username=root --owner=root simple_bank

migrateup:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up

migratedown:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose down

migrateforce1:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose force 1

dropdb:
	sudo docker exec -it backend-master dropdb --username=root --owner=root simple_bank

sqlc:
	sqlc generate

test_sqlc: startdb
	go test ./db/sqlc

.PHONY: postgres createdb dropdb migrateup migratedown sqlc test_sqlc postgres_start
