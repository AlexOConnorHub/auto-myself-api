.PHONY: migrate seed test dev up restart logs  clean run 

ifneq (,$(filter $(MAKECMDGOALS),migrate seed))
  PROVIDED_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  # ...and turn them into do-nothing targets
  $(eval $(PROVIDED_ARGS):;@:)
endif

migrate:
	docker compose -f compose-utilities.yml -f compose.yml run --rm migrate $(PROVIDED_ARGS)

seed:
	docker compose -f compose-utilities.yml -f compose.yml run --rm seed $(PROVIDED_ARGS)

test:
	docker compose -f compose-utilities.yml -f compose.yml up test

dev:
	docker compose up --build --watch api

up:
	docker compose up -d

restart:
	docker compose restart api

logs:
	docker compose logs -f api

down:
	docker compose down
