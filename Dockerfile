FROM arm32v6/golang:1.10.1-alpine3.7 AS builder

ENV GOPATH /go
WORKDIR /go/src

RUN mkdir -p /go/src/github.com/cjimti/iotagent
COPY . /go/src/github.com/cjimti/iotagent

RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o /go/bin/agent /go/src/github.com/cjimti/iotwifi/agent.go

FROM arm32v6/alpine:3.7

RUN apk --no-cache add ca-certificates
WORKDIR /

COPY --from=builder /go/bin/agent /agent
ENTRYPOINT ["/agent"]
