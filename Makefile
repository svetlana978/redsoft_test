.PHONY: test migrate-up migrate-down docker-prepare docker-up-service docker-stop-service docker-stop docker-down-v clean proto deps

include .env

test:
	go test -v -cover ./test/unit

migrate-up:
	docker compose run --rm migrate -path=/migrations -database  "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST_DOCKER}:${DB_PORT}/${DB_NAME}?sslmode=disable" up

migrate-down:
	docker compose run --rm migrate -path=/migrations -database  "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST_DOCKER}:${DB_PORT}/${DB_NAME}?sslmode=disable" down

docker-prepare:
	docker-compose up postgres -d

docker-up-service:
	docker-compose up person-service -d

docker-stop-service:
	docker-compose stop person-service

docker-stop:
	docker-compose stop person-service postgres

docker-down-v:
	docker-compose down -v

proto:
	protoc --go_out=. --go-grpc_out=. proto/person.proto

deps:
	go mod download
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
