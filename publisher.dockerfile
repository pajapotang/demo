FROM golang:1.12

RUN mkdir -p /go/src/github.com/pajapotang/demo
ADD . /go/src/github.com/pajapotang/demo
WORKDIR /go/src/github.com/pajapotang/demo
RUN go get -u github.com/golang/dep/cmd/dep
RUN dep ensure -v
RUN go build -o api -i publisher/main.go
CMD ["/go/src/github.com/pajapotang/demo/api"]

EXPOSE 8080