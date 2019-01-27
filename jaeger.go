package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	olog "github.com/opentracing/opentracing-go/log"
	jaeger "github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"
)

func main() {
	cfg := config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LocalAgentHostPort:  "localhost:6831",
			LogSpans:            true,
			BufferFlushInterval: 1 * time.Second,
		},
	}
	closer, err := cfg.InitGlobalTracer(
		"serviceName",
		config.Logger(jaegerlog.StdLogger),
		config.Metrics(metrics.NullFactory),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer closer.Close()

	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	// https://github.com/opentracing/opentracing-go#deserializing-from-the-wire
	ctx, _ := opentracing.GlobalTracer().Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header),
	)
	fmt.Printf("got ctx %#v", ctx)

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
}
