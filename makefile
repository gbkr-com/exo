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

.PHONY: run-example
run-example: export URL = wss://ws-feed.exchange.coinbase.com
run-example: export RATE = 10
run-example: export SYMBOL = ETH-USD
run-example:
	@cd cmd/example && ./example