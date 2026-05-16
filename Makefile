build:
	go build -o ./bin/cy

run-server: build
	./bin/cy server start

run-worker: build
	./bin/cy worker start 