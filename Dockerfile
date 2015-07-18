FROM golang:1.4.2-onbuild

ADD ./feed.go /usr/src/app/feed.go

VOLUME ["/usr/src/app/files"]

EXPOSE 8080