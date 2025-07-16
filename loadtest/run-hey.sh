#!/bin/bash
# Usage: ./run-hey.sh http://localhost:8080/your-endpoint 100 30 50
# Arguments: <url> <rps> <duration_seconds> <concurrency>

URL=${1:-http://146.190.116.201:8080/v1/chat/completions}
RPS=${2:-1000}
DURATION=${3:-50}
CONCURRENCY=${4:-50}

echo "Running hey load test:"
echo "URL: $URL"
echo "RPS: $RPS"
echo "Duration: ${DURATION}s"
echo "Concurrency: $CONCURRENCY"

cat > prompt.json <<EOF
{
    "model": "gpt-3.5-turbo",
    "messages": [
        {
            "role": "user",
            "content": "Write a detailed essay about the history of artificial intelligence, including major milestones, key figures, and future trends. Make it at least 1000 words."
        }
    ],
    "stream": true
}
EOF

hey -z ${DURATION}s -q $RPS -c $CONCURRENCY -m POST -H "Content-Type: application/json" -D prompt.json $URL


# Extract p50 and p95 latency in seconds, then convert to milliseconds
P50_SEC=$(grep "50%" hey_output.txt | awk '{print $3}')
P95_SEC=$(grep "95%" hey_output.txt | awk '{print $3}')

# Convert to milliseconds using bc
P50_MS=$(echo "$P50_SEC * 1000" | bc)
P95_MS=$(echo "$P95_SEC * 1000" | bc)

echo "p50 latency: ${P50_MS} ms"
echo "p95 latency: ${P95_MS} ms"