.PHONY: help
 help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure [y/N] ' && read ans && [ $${ans:N} = y ]

.PHONY: run/api
run/api:
	go run ./cmd/api

.PHONY: db/migrations/new
db/migrations/new:
	@echo 'Creating migration files for ${name}'
	migrate create -seq -ext .sql -dir ./migrations ${name}

.PHONY: db/migrations/up
db/migrations/up: confirm
	@echo 'Running up migrations...'
	migrate -path ./migrations -database ${DSN} up

.PHONY: audit
audit:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

.PHONY: build/api
build/api:
	@echo 'Building cmd/api...'
	go build -ldflags='-s' -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/api ./cmd/api

.PHONY: production/connect
production/connect:
	ssh greenlight@${PROD_IP}

.PHONY: production/deploy/api
production/deploy/api:
	rsync -P ./bin/api greenlight@${PROD_IP}:~
	rsync -rP --delete ./migrations greenlight@${PROD_IP}:~
	rsync -P ./remote/production/api.service greenlight@${PROD_IP}:~
	ssh -t greenlight@${PROD_IP} '\
    migrate -path ~/migrations -database $$DSN up \
    && sudo mv ~/api.service /etc/systemd/system/ \
    && sudo systemctl enable api \
    && sudo systemctl restart api \
    '