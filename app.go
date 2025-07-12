package main

import (
	"context"
	"log"
	"os"

	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/like-mike/relai-gateway/provider"
	"github.com/like-mike/relai-gateway/routes"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func init() {
	os.Setenv("OTEL_SERVICE_NAME", "relai-gateway")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "0.0.0.0:4317")
	os.Setenv("OTEL_TRACES_SAMPLER", "always_on")
	_ = godotenv.Load()
}

func main() {
	tp := initTracer()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	provider, err := provider.NewProviderFromEnv()
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	// Add OpenTelemetry tracing middleware
	app.Use(otelfiber.Middleware())
	// Add Prometheus metrics middleware
	app.Use(routes.PrometheusMiddleware())
	// Add HTTP logging middleware
	app.Use(logger.New())
	// Expose Prometheus metrics at /metrics
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
	app.Get("/health", routes.HealthHandler)
	routes.RegisterRoutes(app, provider)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("Starting server on :" + port)
	log.Fatal(app.Listen(":" + port))
}

func initTracer() *sdktrace.TracerProvider {
	ctx := context.Background()

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "relai-gateway"
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)
	if err != nil {
		log.Fatalf("Failed to create OTLP exporter: %v", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create resource: %v", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp
}
