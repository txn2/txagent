FROM golang:1.9 AS builder

COPY ./ ./src/github.com/cjimti/

RUN go get ...
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o ./bin/agent ./src/main.go


#FROM alpine:latest
#
#RUN mkdir /app
#WORKDIR /app
#
#COPY --from=builder /go/bin/agent .
#CMD ["./agent"]
