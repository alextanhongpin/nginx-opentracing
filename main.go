package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	olog "github.com/opentracing/opentracing-go/log"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/tracecontext"
	"go.opencensus.io/tag"

	"go.opencensus.io/trace"
)

func main() {
	// https://opencensus.io/exporters/supported-exporters/go/jaeger/
	exporter, err := jaeger.NewExporter(jaeger.Options{
		Endpoint:      "http://localhost:14268",
		AgentEndpoint: "http://localhost:6381",
		// The name must be the same as the one set in the
		// jaeger-nginx-config.json
		ServiceName: "nginx", // Defaults to OpenCensus.
	})
	if err != nil {
		log.Fatal(err)
	}
	trace.RegisterExporter(exporter)
	// For development purpose - always sample.
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	client := &http.Client{Transport: &ochttp.Transport{}}
	mux := http.NewServeMux()
	tctx := new(tracecontext.HTTPFormat)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var span *trace.Span
		spanCtx, _ := tctx.SpanContextFromRequest(r)
		// if !ok {
		//         log.Println("not ok")
		//         ctx, span = trace.StartSpan(ctx, "hello")
		// } else {
		ctx, span = trace.StartSpanWithRemoteParent(ctx, "/hello", spanCtx)
		// }

		span.AddAttributes(trace.StringAttribute("visited endpoint", "this endpoint has been visited"))
		defer span.End()

		req, _ := http.NewRequest("GET", "http://localhost:8080/car", nil)
		req = req.WithContext(ctx)
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()

		fmt.Fprintf(w, "hello")
	})
	mux.HandleFunc("/car", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		tagMap := tag.FromContext(ctx)
		fmt.Println("tagmap2", tagMap)
		span := trace.FromContext(ctx)
		fmt.Printf("%#v\n", span)
		if span == nil {
			ctx, span = trace.StartSpan(ctx, "/")
		} else {
			ctx, span = trace.StartSpanWithRemoteParent(ctx, "world", span.SpanContext())
		}
		span.AddAttributes(trace.StringAttribute("hello", "request kade"))
		defer span.End()
		fmt.Fprintf(w, "car")
	})

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := opentracing.GlobalTracer().Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(r.Header),
		)
		span := opentracing.StartSpan(
			"/greetings",
			opentracing.ChildOf(ctx),
		)
		defer span.Finish()
		span.LogFields(
			olog.String("event", "soft error"),
			olog.String("type", "cache timeout"),
			olog.Int("waited.millis", 1500))
		now := time.Now().Format(time.RFC3339)
		fmt.Fprintf(w, now)
	})
	fmt.Println("listening to port *:8080")
	och := &ochttp.Handler{
		Handler: mux, // The handler you'd have used originally
	}
	http.ListenAndServe(":8080", och)
}
