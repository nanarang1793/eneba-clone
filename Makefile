.PHONY: db-up db-down migrate-up migrate-down seed seed-gen db-reset psql api

DB_URL ?= postgres://eneba:eneba@localhost:5432/eneba?sslmode=disable

db-up:            ## start Postgres
	docker compose up -d db

db-down:          ## stop everything (data volume is kept)
	docker compose down

migrate-up:       ## apply all pending migrations
	docker compose run --rm migrate up

migrate-down:     ## roll back the most recent migration
	docker compose run --rm migrate down 1

seed-gen:         ## regenerate backend/seed/seed.sql from the Go generator
	cd backend && go run ./cmd/seedgen > seed/seed.sql

seed:             ## load seed data into the running database
	docker compose run --rm seed

db-reset:         ## drop all data, re-migrate, re-seed
	docker compose down -v
	docker compose up -d db
	docker compose run --rm migrate up
	docker compose run --rm seed

psql:             ## open a psql shell
	docker compose exec db psql -U eneba -d eneba

api:              ## run the Go API locally
	cd backend && DATABASE_URL="$(DB_URL)" go run ./cmd/api
