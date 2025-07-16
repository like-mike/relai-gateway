FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy the dummy backend source file from loadtest/
COPY app.go ./

RUN go build -o dummy-backend app.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/dummy-backend .

EXPOSE 2000

CMD ["./dummy-backend"]