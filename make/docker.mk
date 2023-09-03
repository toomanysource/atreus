##@ Cli Development

# ================================================
# Public Commands:
# ================================================

.PHONY: docker-compose-up
docker-compose-up: ## Run docker-compose up -d
docker-compose-up: docker.compose.up

.PHONY: docker-compose-down
docker-compose-down: ## Run docker-compose down
docker-compose-down: docker.compose.down

# ================================================
# Private Commands:
# ================================================

.PHONY: docker.compose.up
docker.compose.up:
	@echo "======> Running Docker compose up"
	@docker-compose -f docker/service/docker-compose.yaml up -d

.PHONY: docker.compose.down
docker.compose.down:
	@echo "======> Running Docker compose down"
	@docker-compose -f docker/service/docker-compose.yaml down