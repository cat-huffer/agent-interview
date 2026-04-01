// Package trace 提供基于 OpenTelemetry 的链路追踪能力。
package trace

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TracerProvider 追踪提供者的封装
type TracerProvider struct {
	serviceName string
}

// InitTracer 初始化全局链路追踪。
// 实际生产环境中应配置 OTLP Exporter 将 Span 导出到 Jaeger/Tempo 等后端。
func InitTracer(serviceName string) (*TracerProvider, error) {
	// 实际实现示例：
	// exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(...))
	// tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), ...)
	// otel.SetTracerProvider(tp)

	return &TracerProvider{serviceName: serviceName}, nil
}

// Shutdown 优雅关闭追踪器，确保所有 Span 都已导出
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	// 实际实现: return tp.provider.Shutdown(ctx)
	return nil
}

// StartSpan 创建一个新的 Span 并返回带 Span 的 Context。
// 调用方必须在完成后调用 EndSpan。
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := otel.Tracer("ai-agent-go")
	ctx, span := tracer.Start(ctx, name,
		trace.WithAttributes(attrs...),
	)
	return ctx, span
}

// AddEvent 在当前 Span 上添加事件
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SetError 在当前 Span 上记录错误
func SetError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() && err != nil {
		span.RecordError(err)
	}
}

// TraceID 从 Context 中提取 TraceID 字符串
func TraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// SpanID 从 Context 中提取 SpanID 字符串
func SpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().SpanID().String()
}

// WrapError 为错误添加追踪信息
func WrapError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	traceID := TraceID(ctx)
	if traceID != "" {
		return fmt.Errorf("[trace=%s] %w", traceID, err)
	}
	return err
}
