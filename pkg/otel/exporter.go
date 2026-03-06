package otel

import (
	"context"
	"crypto/tls"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const (
	serviceName  = "lite-api"
	otlpEndpoint = "localhost:4318"
)

func SetupTracer(disabled bool) (func(context.Context) error, error) {
	ctx := context.Background()
	return InstallExportPipeline(ctx, disabled)
}

func Resource() *resource.Resource {
	// Defines resource with service name, version, and environment.
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
		semconv.ServiceVersionKey.String(os.Getenv("VERSION")),
		attribute.String("environment", os.Getenv("ENV")),
	)
}

func InstallExportPipeline(ctx context.Context, disabled bool) (func(context.Context) error, error) {
	if disabled {
		tracerProvider := trace.NewTracerProvider()
		otel.SetTracerProvider(tracerProvider)

		return tracerProvider.Shutdown, nil
	}

	var tlsOption otlptracehttp.Option
	if os.Getenv("ENV") == "dev" {
		tlsOption = otlptracehttp.WithInsecure()
	} else {
		tlsOption = otlptracehttp.WithTLSClientConfig(&tls.Config{})
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(otlpEndpoint),
		tlsOption,
	)
	if err != nil {
		return nil, err
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(Resource()),
	)
	otel.SetTracerProvider(tracerProvider)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tracerProvider.Shutdown, nil
}
