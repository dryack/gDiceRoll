FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /gDiceRoll ./cmd/server

# Install migrate tool
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

COPY --from=builder /gDiceRoll .
COPY --from=builder /app/web /root/web
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /app/core/db/migrations /migrations
COPY --from=builder /app/start.sh /start.sh

RUN chmod +x /start.sh


CMD ["/start.sh"]
