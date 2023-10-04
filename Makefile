m.PHONY: all
all: validate test clean

## Run validates
.PHONY: validate
validate:
	golangci-lint run

## Run tests
.PHONY: test
test: test-start-stack
test:
	go test -v -race ${TEST_ARGS} ./...

## Launch docker stack for test
.PHONY: test-start-stack
test-start-stack:
	docker-compose -f script/docker-compose.yml up --wait

	PORT=26379 envsubst < ./script/conf/sentinel_template.conf > ./script/conf/sentinel1.conf
	PORT=36379 envsubst < ./script/conf/sentinel_template.conf > ./script/conf/sentinel2.conf
	PORT=46379 envsubst < ./script/conf/sentinel_template.conf > ./script/conf/sentinel3.conf
	docker-compose -f script/docker-compose-sentinel.yml up --wait

## Clean local data
.PHONY: clean
clean:
	docker-compose -f script/docker-compose.yml down
	docker-compose -f script/docker-compose-sentinel.yml down
	$(RM) goverage.report $(shell find . -type f -name *.out)
