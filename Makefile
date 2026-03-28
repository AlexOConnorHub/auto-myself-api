.PHONY: migrate seed test dev start restart logs stop clean run

ifneq (,$(filter $(MAKECMDGOALS),migrate seed))
  PROVIDED_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  # ...and turn them into do-nothing targets
  $(eval $(PROVIDED_ARGS):;@:)
endif

test:
	docker compose -f compose-utilities.yml -f compose.yml up test
