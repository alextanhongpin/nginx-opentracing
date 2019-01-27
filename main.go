package main

import (
	"fmt"
	"log"
	"net/http"

	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
)

func main() {
	// cfg := jaegercfg.Configuration{
	//         Sampler: &jaegercfg.SamplerConfig{
	//                 Type:  jaeger.SamplerTypeConst,
	//                 Param: 1,
	//         },
	//         Reporter: &jaegercfg.ReporterConfig{
	//                 LogSpans: true,
	//         },
	// }
	// closer, err := cfg.InitGlobalTracer(
	//         "serviceName",
	//         jaegercfg.Logger(jaegerlog.StdLogger),
	//         jaegercfg.Metrics(metrics.NullFactory),
	// )
	// if err != nil {
	//         log.Fatal(err)
	// }
	// defer closer.Close()
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
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		span := trace.FromContext(ctx)
		fmt.Printf("%#v", span)
		if span == nil {
			ctx, span = trace.StartSpan(ctx, "http/something")
		} else {
			ctx, span = trace.StartSpanWithRemoteParent(ctx, "hello", span.SpanContext())
		}

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

		span := trace.FromContext(ctx)
		fmt.Printf("%#v", span)
		if span == nil {
			ctx, span = trace.StartSpan(ctx, "http/something")
		} else {
			ctx, span = trace.StartSpanWithRemoteParent(ctx, "hello", span.SpanContext())
		}
		span.AddAttributes(trace.StringAttribute("hello", "request kade"))
		defer span.End()
		fmt.Fprintf(w, "car")
	})
	fmt.Println("listening to port *:8080")
	http.ListenAndServe(":8080", mux)
}
