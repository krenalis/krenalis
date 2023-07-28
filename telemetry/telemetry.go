//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package telemetry enables sending telemetry data to the OpenTelemetry
// Collector on "localhost:4317". This, in turn, should be configured to send
// traces and logs to Jaeger and metrics to Prometheus.
package telemetry

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	_trace "go.opentelemetry.io/otel/trace"
)

const (
	tracerName     = "ChichiTracer"
	meterName      = "ChichiMeter"
	otlEndpoint    = "localhost:4317"
	serviceNameKey = "Chichi"
)

// Init initializes the telemetry.
// The context ctx is kept, among other things, to perform the shutdown of the
// telemetry when the context is cancelled.
//
// When the telemetry is initialized, traces and logs are sent to Jaeger while
// metrics are sent to Prometheus; this requires the brokerage of the Open
// Telemetry Collector, which should listens on "localhost:4317".
//
// Init should be called just once.
func Init(ctx context.Context) error {

	if telemetryEnabled {
		panic("telemetry already initialized")
	}

	// Init the TracerProvider.
	{
		exporter, err := otlptracegrpc.New(
			ctx,
			otlptracegrpc.WithEndpoint(otlEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return err
		}
		resource := resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceNameKey),
		)
		tracerProvider := trace.NewTracerProvider(
			trace.WithBatcher(exporter),
			trace.WithResource(resource),
		)
		otel.SetTracerProvider(tracerProvider)
		otel.SetTextMapPropagator(
			// TODO: investigate on the possibilities of this propagator.
			propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
			),
		)
	}

	// Init the MeterProvider.
	{
		exporter, err := otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(otlEndpoint),
			otlpmetricgrpc.WithInsecure())
		if err != nil {
			return err
		}
		resource := resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceNameKey),
		)
		reader := metric.NewPeriodicReader(
			exporter,
			metric.WithInterval(1*time.Second),
		)
		meterProvider := metric.NewMeterProvider(
			metric.WithResource(resource),
			metric.WithReader(reader),
		)
		otel.SetMeterProvider(meterProvider)
	}

	telemetryEnabled = true

	return nil
}

// IncrementCounter increments the int64 counter named name with the given
// increment quantity.
//
// If the telemetry has not been initialized with the Init function, then this
// function is a no-op.
func IncrementCounter(ctx context.Context, name string, incr int64) {
	if !telemetryEnabled {
		return
	}
	// TODO(Gianluca): investigate on other ways to handle a not-responding
	// collector instead of launching a goroutine to make IncrementCounter
	// not-blocking.
	go func() {
		counter, err := otel.Meter(meterName).Int64Counter(name)
		if err != nil {
			log.Printf("[info] IncrementCounter: %s", err)
		}
		counter.Add(ctx, incr)
	}()
}

// TraceSpan traces a span. Returns a context and a Span, on which the method
// End should be called to indicate the Span ending.
//
// TraceSpan accepts a variadic args argument which is a key-value pair sequence
// of attributes that will be added to the span. The even arguments - starting
// from 0 - are the keys and must be strings, while the odds arguments are the
// values associated to the preceding key. Accepted value types are: int,
// float64, string, bool and fmt.Stringer.
//
// An example usage:
//
//	ctx, span := telemetry.TraceSpan(ctx, "apis.Account", "account_id", id)
//	defer span.End()
//
// If the telemetry has not been initialized with the Init function, then this
// function does nothing and returns ctx itself a nil *Span, on which is safe to
// call methods that won't panic.
func TraceSpan(ctx context.Context, name string, args ...any) (context.Context, *Span) {
	if !telemetryEnabled {
		return ctx, nil
	}
	if len(args) == 0 {
		ctx, span := otel.Tracer(tracerName).Start(ctx, name)
		return ctx, &Span{span: span}
	}
	ctx, span := otel.Tracer(tracerName).Start(ctx, name, keyValuePairsToOptions(args))
	return ctx, &Span{span: span}
}

// Span represents a tracer span. A new span can be created with TraceSpan.
type Span struct {
	span _trace.Span
}

// Log logs a message on this span.
//
// Accepts a variadic args argument which is a key-value pair sequence of
// attributes that will be added to the log message. The even arguments -
// starting from 0 - are the keys and must be strings, while the odds arguments
// are the values associated to the preceding key. Accepted value types are:
// int, float64, string, bool and fmt.Stringer.
//
// Example usage:
//
//	span.Log("sum completed", "x", x, "y", y)
//
// If the telemetry has not been initialized with the Init function, then this
// method is a no-op.
func (s *Span) Log(msg string, args ...any) {
	if !telemetryEnabled {
		return
	}
	if len(args) > 0 {
		s.span.AddEvent(msg, keyValuePairsToOptions(args))
	} else {
		s.span.AddEvent(msg)
	}
}

// End ends the span.
func (s *Span) End() {
	if !telemetryEnabled {
		return
	}
	s.span.End()
}

var telemetryEnabled = false

func keyValuePairsToOptions(keyValuePairs []any) _trace.SpanStartEventOption {
	if len(keyValuePairs) == 0 {
		return nil
	}
	if len(keyValuePairs)%2 != 0 {
		panic("key-value pairs must contain an even number of elements")
	}
	attributes := make([]attribute.KeyValue, len(keyValuePairs)/2)
	for n := 0; n < len(keyValuePairs)/2; n++ {
		kIndex := n * 2
		vIndex := n*2 + 1
		key, ok := keyValuePairs[kIndex].(string)
		if !ok {
			panic("key must be a string")
		}
		var attr attribute.KeyValue
		switch v := keyValuePairs[vIndex].(type) {
		case int:
			attr = attribute.Int(key, v)
		case float64:
			attr = attribute.Float64(key, v)
		case string:
			attr = attribute.String(key, v)
		case bool:
			attr = attribute.Bool(key, v)
		case fmt.Stringer:
			attr = attribute.String(key, v.String())
		default:
			panic(fmt.Sprintf("unsupported type %T", v))
		}
		attributes[n] = attr
	}
	return _trace.WithAttributes(attributes...)
}
