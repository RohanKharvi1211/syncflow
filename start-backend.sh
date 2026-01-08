#!/bin/bash

echo "Starting Backend Server..."
cd "$(dirname "$0")/backend"
go run cmd/server/main.go


