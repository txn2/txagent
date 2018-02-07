FROM alpine
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY iotagent /iotagent
CMD ["/iotagent"]