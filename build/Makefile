ENV_FILE := .env
CONFIG_OUTPUT := ./configs/config.yaml
CONFIG_TEMPLATE := config_template.txt
TERN_CONFIG_OUTPUT := ./db/tern.conf
TERN_CONFIG_TEMPLATE := tern_conf_template.txt

ifneq (,$(wildcard $(ENV_FILE)))
ENV_VARS := $(shell grep -vE '^\s*#' $(ENV_FILE) | grep -E '^[A-Za-z0-9_]+=' | sed 's/=.*//')
$(foreach var,$(ENV_VARS),\
  $(if $(value $(var)),,$(eval export $(var)=$(shell grep -m1 "^$(var)=" $(ENV_FILE) | cut -d= -f2-)))\
)
endif

.PHONY: generate

config:
	@echo "Generating $(CONFIG_OUTPUT) from $(CONFIG_TEMPLATE)..."
	@export $$(cat $(ENV_FILE) | sed 's/ *= */=/' | grep -v '^#') && \
	envsubst < $(CONFIG_TEMPLATE) > $(CONFIG_OUTPUT)
	@echo "Done: $(CONFIG_OUTPUT) created."

tern:
	@echo "Generating $(TERN_CONFIG_OUTPUT) from $(TERN_CONFIG_TEMPLATE)..."
	@export $$(cat $(ENV_FILE) | sed 's/ *= */=/' | grep -v '^#') && \
	envsubst < $(TERN_CONFIG_TEMPLATE) > $(TERN_CONFIG_OUTPUT)
	@echo "Done: $(TERN_CONFIG_OUTPUT) created."

generate: config tern