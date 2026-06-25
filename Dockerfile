# syntax=docker/dockerfile:1.7
#
# Build with the sibling go-infra repository as a named context:
#   docker build --build-context go-infra=../go-infra -t buy-ticket:local .

FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY --from=go-infra . /src/go-infra
COPY go.mod go.sum ./

RUN go mod edit -replace=github.com/austin72905/go-infra=/src/go-infra
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
