YOUTRACK_IMAGE ?= jetbrains/youtrack:2026.2.19656
YOUTRACK_PORT ?= 8089
YOUTRACK_CONTAINER ?= terraform-provider-youtrack-test
YOUTRACK_LOCAL_URL = http://localhost:$(YOUTRACK_PORT)
TOKEN_FILE = .youtrack-token

.PHONY: build test testacc docs youtrack-up youtrack-token youtrack-down

build:
	go build ./...

test:
	go test ./...

testacc: youtrack-token
	TF_ACC=1 YOUTRACK_URL=$(YOUTRACK_LOCAL_URL) YOUTRACK_TOKEN=$$(cat $(TOKEN_FILE)) go test ./internal/provider -run TestAcc -count=1 -v

docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest generate --provider-name youtrack

youtrack-up:
	@if [ -z "$$(docker ps -q -f name=^$(YOUTRACK_CONTAINER)$$)" ]; then \
		docker run --rm -v $(YOUTRACK_CONTAINER)-data:/opt/youtrack/data -v $(YOUTRACK_CONTAINER)-conf:/opt/youtrack/conf \
			$(YOUTRACK_IMAGE) configure -J-Ddisable.configuration.wizard.on.clean.install=true --base-url=$(YOUTRACK_LOCAL_URL); \
		docker run -d --name $(YOUTRACK_CONTAINER) -p $(YOUTRACK_PORT):8080 \
			-v $(YOUTRACK_CONTAINER)-data:/opt/youtrack/data -v $(YOUTRACK_CONTAINER)-conf:/opt/youtrack/conf $(YOUTRACK_IMAGE); \
	fi
	@echo "Waiting for YouTrack on $(YOUTRACK_LOCAL_URL)"
	@until curl -sf -o /dev/null -u admin:admin $(YOUTRACK_LOCAL_URL)/api/users/me; do sleep 5; done

youtrack-token: youtrack-up
	@if [ ! -s $(TOKEN_FILE) ]; then \
		service=$$(curl -sf -u admin:admin "$(YOUTRACK_LOCAL_URL)/hub/api/rest/services?fields=id,applicationName" \
			| python3 -c 'import json,sys;print(next(s["id"] for s in json.load(sys.stdin)["services"] if s["applicationName"]=="YouTrack"))'); \
		curl -sf -u admin:admin -H 'Content-Type: application/json' "$(YOUTRACK_LOCAL_URL)/hub/api/rest/users/me/permanenttokens?fields=token" \
			-d "{\"name\":\"terraform-acceptance\",\"scope\":[{\"id\":\"0-0-0-0-0\"},{\"id\":\"$$service\"}]}" \
			| python3 -c 'import json,sys;print(json.load(sys.stdin)["token"])' > $(TOKEN_FILE); \
	fi

youtrack-down:
	-docker rm -f $(YOUTRACK_CONTAINER)
	-docker volume rm $(YOUTRACK_CONTAINER)-data $(YOUTRACK_CONTAINER)-conf
	rm -f $(TOKEN_FILE)
