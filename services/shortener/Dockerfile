FROM golang:1.25-alpine AS build

WORKDIR /src/services/shortener

COPY services/shortener/go.mod services/shortener/go.sum ./
RUN go mod download

COPY services/shortener ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/shortener ./cmd/shortener

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=build /out/shortener /usr/local/bin/shortener

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/shortener"]
