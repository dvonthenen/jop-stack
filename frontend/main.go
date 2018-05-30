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

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
)

var (
	hostIP     string
	listenIP   string
	listenPort string
	backendURL string
	httpAddr   string
	jaegerAddr string
)

const (
	defaultJaegerPort = "6831"
	defaultHostIP     = "localhost"
	defaultListenPort = "80"
	defaultListenIP   = "0.0.0.0"
	defaultBackendURL = "http://backend:80"
)

var (
	httpRequestsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "cloud_native_app",
			Subsystem: "frontend",
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

	hostIP = env("HOST_IP", defaultHostIP)
	listenIP = env("LISTEN_IP", defaultListenIP)
	listenPort = env("LISTEN_PORT", defaultListenPort)
	backendURL = env("BACKEND_URI", defaultBackendURL)

	log.Info("hostIP:", hostIP)
	log.Info("listenIP:", listenIP)
	log.Info("listenPort:", listenPort)
	log.Info("backendURL:", backendURL)

	jaegerAddr := net.JoinHostPort(hostIP, defaultJaegerPort)
	httpAddr := net.JoinHostPort(listenIP, listenPort)

	// Jaeger tracer can be initialized with a transport that will
	// report tracing Spans to a Zipkin backend
	transport, err := jaeger.NewUDPTransport(jaegerAddr, 0)
	if err != nil {
		log.Fatal("unable to create Zipkin tracer:", err)
	}

	tracer, _ := jaeger.NewTracer(
		"frontend",
		jaeger.NewConstSampler(true), // sample all traces
		jaeger.NewRemoteReporter(transport),
		jaeger.TracerOptions.ZipkinSharedRPCSpan(true),
	)
	// Close the tracer to guarantee that all spans that could
	// be still buffered in memory are sent to the tracing backend
	//defer closer.Close()

	opentracing.InitGlobalTracer(tracer)

	http.HandleFunc("/", handler)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(httpAddr, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Info("StartSpan")
	span := opentracing.StartSpan("frontend")
	span.SetTag("service", "frontend")
	defer span.Finish()

	log.Info("NewRequest")
	httpRequest, err := http.NewRequest(http.MethodGet, backendURL, nil)
	if err != nil {
		log.Error("error creating new backend HTTP request:", err)
		httpRequestsCounter.WithLabelValues(r.Method, "500").Inc()
		http.Error(w, "service unavailable", 500)
		return
	}

	log.Info("StartSpan")
	childSpan := opentracing.StartSpan("frontend", opentracing.ChildOf(span.Context()))
	opentracing.GlobalTracer().Inject(
		childSpan.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(httpRequest.Header),
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

	httpRequestsCounter.WithLabelValues(r.Method, "200").Inc()
}
