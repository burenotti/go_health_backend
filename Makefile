IMAGE = health_backend
IMAGE_TAG = latest
COMPOSE_FILE = compose.yaml
COMPOSE = docker compose -f $(COMPOSE_FILE)

logs:
	$(COMPOSE) logs server --follow

up:
	$(COMPOSE) up -d

up_watch:
	$(COMPOSE) up -d --watch

docker_build:
	$(COMPOSE) build

docker_build_no_cache:
	$(COMPOSE) build --no-cache

build:
	CGO_ENABLED=0 go build -ld -o ./bin/server cmd/app/main.go

test:
	CGO_ENABLED=0 go test -race -coverprofile=coverage.out ./...

coverage: test
	go tool -html=coverage.out -o coverage.html
