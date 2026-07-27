// Package tracing — OpenTelemetry distributed tracing sozlamasi.
package tracing

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Init OpenTelemetry tracer provider'ni sozlaydi.
// OTEL_EXPORTER_OTLP_ENDPOINT o'rnatilmagan bo'lsa — no-op (global default provider),
// qo'shimcha overhead yo'q. Shutdown funksiyasini qaytaradi (server o'chganда flush).
func Init(ctx context.Context, serviceName, env string) (func(context.Context) error, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return func(context.Context) error { return nil }, nil // tracing o'chiq
	}

	exp, err := otlptracehttp.New(ctx) // endpoint OTEL_EXPORTER_OTLP_ENDPOINT env'dan
	if err != nil {
		return nil, err
	}
	res, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("deployment.environment", env),
	))
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
