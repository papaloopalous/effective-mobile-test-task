FROM golang:1.24-alpine AS builder
COPY . /go/src/app
WORKDIR /go/src/app
RUN go build -o app ./cmd/

FROM alpine AS runtime
WORKDIR /app
COPY --from=builder /go/src/app/app /app/
RUN chmod +x ./app
EXPOSE 8080/tcp
ENTRYPOINT ./app