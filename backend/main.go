// Copyright 2016 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"

	log "github.com/Sirupsen/logrus"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	/*
		jaeger "github.com/uber/jaeger-client-go"
		"github.com/uber/jaeger-client-go/zipkin"
	*/
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"
)

var (
	listenIP   string
	listenPort string
	httpAddr   string
	//tracer     opentracing.Tracer
)

const (
	defaultListenPort = "80"
	defaultListenIP   = "0.0.0.0"
)

var (
	httpRequestsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "cloud_native_app",
			Subsystem: "backend",
			Name:      "http_requests_total",
			Help:      "Number of HTTP requests",
		},
		[]string{"method", "status"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsCounter)
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func env(key, defaultValue string) (value string) {
	if value = os.Getenv(key); value == "" {
		value = defaultValue
	}
	return
}

func main() {
	var err error

	listenIP = env("LISTEN_IP", defaultListenIP)
	listenPort = env("LISTEN_PORT", defaultListenPort)

	log.Info("listenIP:", listenIP)
	log.Info("listenPort:", listenPort)

	httpAddr := net.JoinHostPort(listenIP, listenPort)

	// Jaeger tracer can be initialized with a transport that will
	// report tracing Spans to a Zipkin backend
	/*
		transport, err := jaeger.NewUDPTransport("localhost:6832", 0)
		if err != nil {
			log.Fatal("unable to create Zipkin tracer:", err)
		}

		// create Jaeger tracer
		zipkinPropagator := zipkin.NewZipkinB3HTTPHeaderPropagator()
		injector := jaeger.TracerOptions.Injector(opentracing.HTTPHeaders, zipkinPropagator)
		extractor := jaeger.TracerOptions.Extractor(opentracing.HTTPHeaders, zipkinPropagator)

		tracer, _ := jaeger.NewTracer(
			"backend",
			jaeger.NewConstSampler(true), // sample all traces
			jaeger.NewRemoteReporter(transport),
			injector,
			extractor,
			jaeger.TracerOptions.ZipkinSharedRPCSpan(true),
		)
		// Close the tracer to guarantee that all spans that could
		// be still buffered in memory are sent to the tracing backend
		//defer closer.Close()

		opentracing.InitGlobalTracer(tracer)
	*/

	// Sample configuration for testing. Use constant sampling to sample every trace
	// and enable LogSpan to log every span via configured Logger.
	cfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans: true,
		},
	}

	// Example logger and metrics factory. Use github.com/uber/jaeger-client-go/log
	// and github.com/uber/jaeger-lib/metrics respectively to bind to real logging and metrics
	// frameworks.
	jLogger := jaegerlog.StdLogger
	jMetricsFactory := metrics.NullFactory

	// Initialize tracer with a logger and a metrics factory
	closer, err := cfg.InitGlobalTracer(
		"serviceName",
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),
	)
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
	defer closer.Close()

	http.HandleFunc("/", handler)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(httpAddr, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	var span opentracing.Span

	log.Info("Extract")
	spanContext, err := opentracing.GlobalTracer().Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header),
	)

	if err != nil {
		log.Error("error extracting opentracing headers:", err)
		httpRequestsCounter.WithLabelValues(r.Method, "500").Inc()
		http.Error(w, "service unavailable", 500)
		return
	}

	log.Info("StartSpan")
	span = opentracing.StartSpan("backend", ext.RPCServerOption(spanContext))
	defer span.Finish()

	log.Info("NewRequest")
	httpRequest, err := http.NewRequest(http.MethodGet, "http://www.google.com", nil)
	if err != nil {
		log.Error("error creating new HTTP request:", err)
		httpRequestsCounter.WithLabelValues(r.Method, "500").Inc()
		http.Error(w, "service unavailable", 500)
		return
	}

	log.Info("StartSpan")
	childSpan := opentracing.StartSpan(
		"get google.com",
		opentracing.ChildOf(span.Context()),
	)
	defer childSpan.Finish()

	log.Info("Do")
	resp, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		log.Error("error running http request:", err)
		httpRequestsCounter.WithLabelValues(r.Method, "500").Inc()
		http.Error(w, "service unavailable", 500)
		return
	}
	defer resp.Body.Close()

	byHTTP, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("error running ReadAll:", err)
		httpRequestsCounter.WithLabelValues(r.Method, "500").Inc()
		http.Error(w, "service unavailable", 500)
		return
	}
	fmt.Fprintf(w, string(byHTTP))

	log.Info("WithLabelValues")
	httpRequestsCounter.WithLabelValues(r.Method, "200").Inc()
}
