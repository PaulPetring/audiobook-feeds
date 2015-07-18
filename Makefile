all: remove build run

.PHONY: build run

build:
	docker build -t audiobook-feeds .

run:
	sh -c 'docker run --rm -v `pwd`:/usr/src/myapp -w /usr/src/myapp -it --rm -p 8080:8080 --name audiobook-feeds audiobook-feeds go run feed.go'

run-no-docker:
	sh -c 'go get . && go run feed.go'

remove:
	docker stop feed-go && docker rm audiobook-feeds
