#!/bin/sh
echo "Contents of /migrations directory:"
ls -la /migrations
echo "File contents:"
cat /migrations/*
echo
echo "Running migrations..."
echo "Using database URL: postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/gdiceroll?sslmode=disable"
migrate -path=/migrations -database="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/gdiceroll?sslmode=disable" up