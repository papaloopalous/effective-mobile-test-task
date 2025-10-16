ENV_FILE := .env
CONFIG_OUTPUT := ./configs/config.yaml
CONFIG_TEMPLATE := config_template.txt
TERN_CONFIG_OUTPUT := ./db/tern.conf
TERN_CONFIG_TEMPLATE := tern_conf_template.txt

.PHONY: generate

config:
	@echo "Generating $(CONFIG_OUTPUT) from $(CONFIG_TEMPLATE)..."
	@envsubst < $(CONFIG_TEMPLATE) > $(CONFIG_OUTPUT)
	@echo "Done: $(CONFIG_OUTPUT) created."

tern:
	@echo "Generating $(TERN_CONFIG_OUTPUT) from $(TERN_CONFIG_TEMPLATE)..."
	@envsubst < $(TERN_CONFIG_TEMPLATE) > $(TERN_CONFIG_OUTPUT)
	@echo "Done: $(TERN_CONFIG_OUTPUT) created."

generate: config tern