PHONY: up-db run-service

up-db:
	docker-compose up -d mysql --remove-orphans

run-service:
	go run cmd/server/main.go

.PHONY: all clean

all: up-db run-service

clean:
	docker-compose down -v
	rm -f split-expense
