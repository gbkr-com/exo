.PHONY: test
test:
	@go test ./... -cover

.PHONY: godoc
godoc:
	@~/go/bin/godoc -http=:8080

.PHONY: browse
browse:
	@open http://localhost:8080/pkg/github.com/gbkr-com/exo