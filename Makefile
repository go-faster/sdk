test:
	@./go.test.sh

coverage:
	@./go.coverage.sh

test_fast:
	go test ./...

tidy:
	go mod tidy

.PHONY: tidy coverage test test_fast