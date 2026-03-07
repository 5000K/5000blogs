FROM golang:latest AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /5000blogs .

FROM alpine:latest

COPY --from=builder /5000blogs /5000blogs
COPY template/ /static/

WORKDIR /

EXPOSE 8080
ENTRYPOINT ["/5000blogs"]
