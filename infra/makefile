COMPOSE := docker compose -f docker-compose.yml

.DEFAULT_GOAL := up
.PHONY: up down reset migrate ps logs psql redis-cli

# Start postgres + redis in the background
up:
	$(COMPOSE) up -d --wait

# Apply pending migrations from service-share-link/migrations
migrate:
	$(MAKE) -C ../service-share-link migrate-up

# Stop containers (data volumes are kept)
down:
	$(COMPOSE) down

# Stop containers and delete data volumes
reset:
	$(COMPOSE) down -v

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs -f

psql:
	$(COMPOSE) exec postgres psql -U postgres -d share_link

redis-cli:
	$(COMPOSE) exec redis redis-cli
