.PHONY: test
test:
	@go test ./... -cover

.PHONY: godoc
godoc:
	@~/go/bin/godoc -http=:8080

.PHONY: browse
browse:
	@open http://localhost:8080/pkg/github.com/gbkr-com/exo


.PHONY: build
build:
	@cd cmd/example && go build
	@cd cmd/paper && go build

.PHONY: run-example
run-example: export URL = wss://ws-feed.exchange.coinbase.com
run-example: export RATE = 10
run-example: export HTTP = :8080
run-example: export REDIS = localhost:6379
run-example: export KEY = :hash:orders
run-example:
	@cd cmd/example && ./example

.PHONY: run-paper
run-paper:
	@cd cmd/paper && ./paper
