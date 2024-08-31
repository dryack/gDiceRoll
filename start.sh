#!/bin/sh

# Run migrations
migrate -path=../migrations -database="postgres://${GDICEROLL_POSTGRES_USER}:${GDICEROLL_POSTGRES_PASSWORD}@${GDICEROLL_POSTGRES_HOST}:${GDICEROLL_POSTGRES_PORT}/${GDICEROLL_POSTGRES_DBNAME}?sslmode=disable" up

# Start the application
./gDiceRoll