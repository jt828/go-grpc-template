.PHONY: proto test-unit test-integration migration docker-build

proto:
	protoc -I=proto/v1 \
		--go_out=proto --go_opt=paths=source_relative \
		--go-grpc_out=proto --go-grpc_opt=paths=source_relative \
		proto/v1/*.proto

test-unit:
	go test ./test/unit/ -v

test-integration:
	go test ./test/integration/ -v -timeout 120s

migration:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

docker-build:
	@read -p "Version (vx.x.x): " version; \
	docker buildx build --platform linux/amd64 -t go-grpc-template:$$version .
