#!/bin/bash
set -e

echo "--- 1. Restoring workspace/fastssz to clean state ---"
cd workspace/fastssz
git restore .
cd ../..

echo "--- 2. Running Instrumentor ---"
go run cmd/instrumentor/main.go -dir ./workspace/fastssz

echo "--- 3. Running CSVV Experiment ---"
go run cmd/csvv/main.go
