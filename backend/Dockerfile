FROM golang:1.25-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache wget
COPY --from=build /out/api ./api
COPY internal/enrichment/fixtures/enrichment_fixtures.json ./internal/enrichment/fixtures/enrichment_fixtures.json

EXPOSE 8080
ENTRYPOINT ["./api"]
