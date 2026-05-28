APP := matching-service
CMD := ./cmd/matching-service
BIN_DIR := ./bin
BIN := $(BIN_DIR)/$(APP)
API_PROTO_FILES := $(shell find api -name '*.proto')

.PHONY: all init generate tidy fmt wire proto test integration build run clean
.PHONY: db-init rocketmq-topic rocketmq-topic-list rocketmq-dlq-list rocketmq-replay

all: tidy fmt generate test build

init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/google/wire/cmd/wire@latest
	go mod tidy

generate: proto wire

tidy:
	go mod tidy

fmt:
	gofmt -w cmd internal pkg

wire:
	wire ./cmd/matching-service

proto:
	./scripts/proto/generate.sh $(API_PROTO_FILES)

test:
	go test ./...

integration:
	go test -count=1 -tags=integration ./test/integration -v

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN) $(CMD)

run:
	go run $(CMD) -conf "$${CONF:-./configs/config.yaml}"

db-init:
	./scripts/mysql/init.sh

rocketmq-topic:
	./scripts/rocketmq/create_topics.sh

rocketmq-topic-list:
	./scripts/rocketmq/list_topics.sh

rocketmq-dlq-list:
	./scripts/rocketmq/list_dlq.sh

rocketmq-replay:
	./scripts/rocketmq/replay.sh

clean:
	rm -rf $(BIN_DIR)
