.PHONY: install up down logs ps restart config pull build rebuild reset clean help

install:
	./setup.sh
	docker compose up -d --build

up:
	docker compose up -d

build:
	docker build -t moneromerchant/monero-merchant-backend ./backend
	docker build -t monero-merchant-frontend ./frontend

rebuild:
	$(MAKE) build
	docker compose up -d --force-recreate backend frontend

down:
	docker compose down

logs:
	docker compose logs -f

ps:
	docker compose ps

restart:
	docker compose down
	docker compose up -d

config:
	docker compose config

pull:
	docker compose pull --ignore-buildable

reset:
	docker compose down -v --remove-orphans
	rm -f .env
	sudo rm -rf moneropay
	sudo rm -rf simple-monero-wallet-rpc-docker

clean:
	docker compose down --remove-orphans
	docker builder prune -f

help:
	@echo "Available targets:"
	@echo "  make install   - run setup.sh, build and start services"
	@echo "  make up        - start existing containers"
	@echo "  make build     - build the backend and frontend images from local source"
	@echo "  make rebuild   - build local source and recreate the backend and frontend"
	@echo "  make down      - stop containers"
	@echo "  make logs      - follow container logs"
	@echo "  make ps        - show container status"
	@echo "  make restart   - restart containers"
	@echo "  make config    - show resolved docker compose config"
	@echo "  make pull      - pull remote images"
	@echo "  make reset     - DELETE containers, database/wallet data, .env, and cloned deps"
	@echo "  make clean     - remove containers and prune builder cache; preserve application data"
