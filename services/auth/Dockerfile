FROM golang:1.25-alpine AS build

WORKDIR /src/services/auth

COPY services/auth/go.mod services/auth/go.sum ./
RUN go mod download

COPY services/auth ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/auth ./cmd/auth

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=build /out/auth /usr/local/bin/auth

EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/auth"]
