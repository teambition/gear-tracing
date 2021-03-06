package tracing

import (
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/teambition/gear"
)

// XRequestID is span baggageItem key for "X-Request-ID" header
const XRequestID = "x-request-id"

// New returns a tracing middleware, use this with *gear.Router
func New(opts ...opentracing.StartSpanOption) gear.Middleware {
	return func(ctx *gear.Context) error {
		// Attempt to join a trace by getting trace context from the headers.
		wireContext, err := opentracing.GlobalTracer().Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(ctx.Req.Header))

		// copy opts avoiding append in the same opts each time.
		// ChildOf will ignore the nil wireContext.
		opts := append([]opentracing.StartSpanOption{opentracing.ChildOf(wireContext)}, opts...)

		var operationName string
		router := gear.GetRouterPatternFromCtx(ctx)
		if router != "" {
			operationName = fmt.Sprintf("%s %s", ctx.Method, router)
		} else {
			operationName = fmt.Sprintf("%s %s", ctx.Method, ctx.Req.RequestURI)
		}
		span := opentracing.StartSpan(operationName, opts...)
		if err != nil && err != opentracing.ErrSpanContextNotFound {
			// record the unexpected error when parsing trace from header.
			span.LogFields(log.Error(err))
		}

		ext.SpanKindRPCServer.Set(span)
		ext.HTTPMethod.Set(span, ctx.Method)
		ext.HTTPUrl.Set(span, ctx.Req.RequestURI)
		span.SetTag("http.host", ctx.Host)

		ctx.WithContext(opentracing.ContextWithSpan(ctx.Context(), span))
		ctx.OnEnd(func() {
			code := ctx.Res.Status()
			ext.HTTPStatusCode.Set(span, uint16(code))
			if code >= 400 {
				ext.Error.Set(span, true)
			}
			span.Finish()
		})
		return nil
	}
}
