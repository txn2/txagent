FROM arm32v6/alpine:37
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY iotagent /iotagent
CMD ["/iotagent"]