# Load Testing with hey

## Installation

Download the `hey` binary from [https://github.com/rakyll/hey/releases](https://github.com/rakyll/hey/releases) and place it in your PATH.

On macOS, you can use Homebrew:
```sh
brew install hey
```

## Usage

To test your gateway at a specific RPS and measure latency percentiles, run:

```sh
hey -z 30s -q 100 -c 50 http://localhost:8080/v1/completions
```
- `-z 30s`: test duration (30 seconds)
- `-q 100`: requests per second (RPS)
- `-c 50`: number of concurrent workers

Or use the provided script:
```sh
./run-hey.sh http://localhost:8080/v1/completions 100 30 50
```

## Dummy Backend for Testing

A simple backend is provided in [`loadtest/dummy_backend.go`](loadtest/dummy_backend.go:1):

```sh
go run loadtest/dummy_backend.go
```

This starts a server on port 2000 with a `/v1/completions` POST endpoint returning a static JSON response.

## Output

`hey` will report latency statistics including p50 (median) and p95 (95th percentile) in milliseconds.

## Example Output

```
Latency Distribution:
  50% in 12.3 ms
  95% in 25.7 ms
```

## Interpreting Results and Adjusting RPS

- p50 (median): Half of requests are faster than this latency.
- p95: 95% of requests are faster than this latency.
- Increase RPS (`-q`) to stress test and observe how latency changes.
- Use results to identify bottlenecks and optimize your gateway.
