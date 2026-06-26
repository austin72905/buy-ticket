# syntax=docker/dockerfile:1.7
#
# Build from this repository:
#   docker build -t buy-ticket:local .

FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/buy-ticket .

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/buy-ticket /app/buy-ticket

ENV APP_ENV=dev
EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/app/buy-ticket"]
