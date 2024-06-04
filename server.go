package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/hyperdxio/opentelemetry-go/otelzap"
	"github.com/hyperdxio/opentelemetry-logs-go/exporters/otlp/otlplogs"
	"github.com/hyperdxio/otel-config-go/otelconfig"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	sdk "github.com/hyperdxio/opentelemetry-logs-go/sdk/logs"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/sdk/resource"
)

// configure common attributes for all logs
func newResource() *resource.Resource {
	hostName, _ := os.Hostname()
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceVersion("1.0.0"),
		semconv.HostName(hostName),
	)
}

// attach trace id to the log
func WithTraceMetadata(ctx context.Context, logger *zap.Logger) *zap.Logger {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		// ctx does not contain a valid span.
		// There is no trace metadata to add.
		return logger
	}
	return logger.With(
		zap.String("trace_id", spanContext.TraceID().String()),
		zap.String("span_id", spanContext.SpanID().String()),
	)
}

func main() {
	// Initialize otel config and use it across the entire app
  println("Service starting up")

	otelShutdown, err := otelconfig.ConfigureOpenTelemetry()
	if err != nil {
		log.Fatalf("error setting up OTel SDK - %e", err)
	}
	defer otelShutdown()

	ctx := context.Background()

	// configure opentelemetry logger provider
	logExporter, _ := otlplogs.NewExporter(ctx)
	loggerProvider := sdk.NewLoggerProvider(
		sdk.WithBatcher(logExporter),
	)
	// gracefully shutdown logger to flush accumulated signals before program finish
	defer loggerProvider.Shutdown(ctx)

	// create new logger with opentelemetry zap core and set it globally
	logger := zap.New(otelzap.NewOtelCore(loggerProvider))
	zap.ReplaceGlobals(logger)
	logger.Warn("hello world", zap.String("foo", "bar"))

	http.Handle("/", otelhttp.NewHandler(wrapHandler(logger, ExampleHandler), "example-service"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("** Service Started on Port " + port + " **")
	println("** Service Started on Port " + port + " **")
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Fatal(err.Error())
	}
}

// Use this to wrap all handlers to add trace metadata to the logger
func wrapHandler(logger *zap.Logger, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := WithTraceMetadata(r.Context(), logger)
		logger.Info("request received", zap.String("url", r.URL.Path), zap.String("method", r.Method))
		handler(w, r)
		logger.Info("request completed", zap.String("path", r.URL.Path), zap.String("method", r.Method))
	}
}

func ExampleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	io.WriteString(w, `{"status":"ok"}`)
}
