FROM golang:1.12

RUN mkdir -p /go/src/github.com/pajapotang/demo
ADD . /go/src/github.com/pajapotang/demo
WORKDIR /go/src/github.com/pajapotang/demo
RUN go get -u github.com/golang/dep/cmd/dep
RUN dep ensure -v
RUN go build -o worker -i consumer/main.go
CMD ["/fo/src/github.com/pajapotang/demo/worker"]