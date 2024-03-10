ifneq (,$(wildcard ./.env))
    include .env
	export
endif

DATABASE_URL=postgres://$(DATABASE_USER):$(DATABASE_PASS)@$(DATABASE_HOST):$(DATABASE_PORT)

run:
	go run .

run-release:
	GIN_MODE=release go run .

database-check:
	until nc -z -v -w30 $(DATABASE_HOST) $(DATABASE_PORT); do \
	  sleep 1; \
	done

database-create:
	psql $(DATABASE_URL) -c "CREATE DATABASE $(DATABASE_NAME)"

database-drop:
	psql $(DATABASE_URL) -c "DROP DATABASE $(DATABASE_NAME)" || exit 0

database-migration-up:
	migrate -path migrations/ -database $(DATABASE_URL)/$(DATABASE_NAME)?sslmode=disable -verbose up

database-migration-create:
	migrate create -ext sql -dir migrations -seq $(name)

database-setup:
	make database-create
	make database-migration-up

docker-compose:
	make docker-compose-down
	docker compose up

docker-compose-build:
	make docker-compose-down
	docker compose up --build

docker-compose-down:
	docker stop postgres-15 || exit 0
	docker stop postgres-11 || exit 0
	docker stop postgres || exit 0
	docker compose  down || exit 0
