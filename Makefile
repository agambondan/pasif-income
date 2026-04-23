.PHONY: build run test clean up down restart ps logs api creator clipper web browser-launcher

COMPOSE ?= docker compose
WEB_PORT ?= 13100
API_PORT ?= 18080

build:
	go build -o bin/api cmd/api/main.go

run:
	go run cmd/api/main.go

test:
	go test ./...

clean:
	rm -rf bin/

up:
	$(COMPOSE) up -d --build db minio whisper redis app browser-launcher clipper web

down:
	$(COMPOSE) down

restart:
	$(COMPOSE) up -d --build --force-recreate db minio whisper redis app browser-launcher clipper web

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs -f --tail=100

api:
	$(COMPOSE) up -d --build app

creator:
	$(COMPOSE) --profile manual run --rm creator

clipper:
	$(COMPOSE) up -d --build clipper

web:
	$(COMPOSE) up -d --build web

browser-launcher:
	python3 scripts/browser_launcher.py watch --dir .runtime/browser-launch-requests
