.PHONY: test
test:
	go test -v -tags sqlite .

.PHONY: test-fast
test-fast:
	go test -v .
