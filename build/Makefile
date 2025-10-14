ENV_FILE := .env
CONFIG_OUTPUT := ./configs/config.yaml
CONFIG_TEMPLATE := config_template.txt

.PHONY: generate

generate:
	@echo "Generating $(CONFIG_OUTPUT) from $(CONFIG_TEMPLATE)..."
	@export $$(cat $(ENV_FILE) | sed 's/ *= */=/' | grep -v '^#') && \
	envsubst < $(CONFIG_TEMPLATE) > $(CONFIG_OUTPUT)
	@echo "Done: $(CONFIG_OUTPUT) created."