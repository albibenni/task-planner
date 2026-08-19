.PHONY: build check fmt fmt-check hooks install lint test test-integration

hooks:
	git config core.hooksPath .githooks
	@echo "Git hooks installed for this clone."

build:
	go build -o task-planner ./cmd/task-planner

install:
	go install ./cmd/task-planner
	@echo "Installed task-planner to $$(go env GOPATH)/bin. Ensure that directory is on PATH."

fmt:
	gofmt -w $$(find cmd -name '*.go' -type f)

fmt-check:
	@test -z "$$(gofmt -l $$(find cmd -name '*.go' -type f))"

lint:
	go vet ./...
	golangci-lint run

test:
	go test ./...

test-integration:
	@trap 'docker compose -f docker-compose.integration.yml down --volumes' EXIT; \
	docker compose -f docker-compose.integration.yml up --wait; \
	TEST_DATABASE_URL=postgres://task_planner_test:task_planner_test@127.0.0.1:54329/task_planner_test go test ./...

check: fmt-check lint test
