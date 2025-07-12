#!/bin/bash

# Call the Prometheus /metrics endpoint
curl -X GET "http://localhost:8080/metrics"