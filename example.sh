#!/bin/bash

export OTEL_TRACES_EXPORTER="none"
export OTEL_METRICS_EXPORTER="none"
export OTEL_LOGS_EXPORTER="stderr"

go run ./cmd/sdk-example