run:
	go run ./cmd/api

migrate:
	@echo 'Creating migration files for ${name}'
	migrate create -seq -ext .sql -dir ./migrations ${name}


up:
	@echo 'Running up migrations...'
	migrate -path ./migrations -database postgres://dzenthai:1234@localhost:5432/greenlight?sslmode=disable up