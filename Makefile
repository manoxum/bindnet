.PHONY: build build-dev run run-dev dev watch watch-backend watch-frontend watch-worker watch-dns run-infra run-deploy run-service run-local run-all stop stop-infra stop-deploy stop-service stop-local stop-all publish republish dist check help

.DEFAULT_GOAL := up

.PHONY: up

up:
	@docker compose --env-file .env.main up --build -d

.PHONY: install-dev start-dev check-launcher

install-dev:
	@python3 ./scripts/desktop/install.py

start-dev:
	@python3 ./scripts/desktop/launcher.py

check-launcher:
	@python3 -m unittest discover -s scripts/desktop/tests -p '*_test.py'

build:
	@./bin/promote build

build-dev:
	@./bin/promote build --dev

run-dev: dev

dev:
	@./bin/promote dev

watch:
	@./bin/promote watch

watch-backend:
	@./bin/promote watch backend

watch-frontend:
	@./bin/promote watch frontend

watch-worker:
	@./bin/promote watch worker

watch-dns:
	@./bin/promote watch dns-provider

run:
	@./bin/promote run

run-infra:
	@./bin/promote run infra

run-deploy:
	@./bin/promote run deploy

run-service:
	@./bin/promote run service

run-local:
	@./bin/promote run service

run-all:
	@./bin/promote run *

stop:
	@./bin/promote stop

stop-infra:
	@./bin/promote stop infra

stop-deploy:
	@./bin/promote stop deploy

stop-service:
	@./bin/promote stop service

stop-local:
	@./bin/promote stop service

stop-all:
	@./bin/promote stop *

publish:
	@./bin/promote publish

republish:
	@./bin/promote republish

dist:
	@./bin/promote dist

check:
	@./bin/promote check all

help:
	@echo "Available targets:"
	@echo "  up (padrão)     - Compilar e iniciar com docker compose --env-file .env.main up --build -d"
	@echo "  install-dev     - Instalar atalho do Bindnet no menu do Linux"
	@echo "  start-dev       - Validar, compilar e abrir admin.bindnet.local.com"
	@echo "  check-launcher  - Validar o launcher sem iniciar serviços reais"
	@echo "  dev             - Start dev stack through bin/promote"
	@echo "  watch           - Watch all configured dev services"
	@echo "  run             - Run main compose aggregator"
	@echo "  run-local       - Run service stack with local ports exposed"
	@echo "  run-infra       - Run infra only"
	@echo "  run-service     - Run Bindnet services"
	@echo "  run-deploy      - Run infra + services"
	@echo "  run-all         - Run all compose groups"
	@echo "  stop-all        - Stop all compose groups"
	@echo "  build           - Build images through bin/promote"
	@echo "  publish         - Publish images through bin/promote"
	@echo "  republish       - Republish images through bin/promote"
	@echo "  dist            - Flatten compose includes to deployment/"
	@echo "  check           - Run promote prerequisite checks"
