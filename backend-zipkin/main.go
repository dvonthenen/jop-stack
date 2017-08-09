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
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	listenIP     string
	listenPort   string
	advertisedIP string
	httpAddr     string
	serviceAddr  string
	zipkinURL    string
	tracer       opentracing.Tracer
)

const (
	defaultListenPort = "80"
	defaultListenIP   = "0.0.0.0"
	defaultZipkinURL  = "http://jaeger-collector:9411/api/v1/spans"
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

	advertisedIP = env("POD_IP", "127.0.0.1")
	listenIP = env("LISTEN_IP", defaultListenIP)
	listenPort = env("LISTEN_PORT", defaultListenPort)
	zipkinURL = env("ZIPKIN_URI", defaultListenPort)

	log.Info("advertisedIP:", advertisedIP)
	log.Info("listenIP:", listenIP)
	log.Info("listenPort:", listenPort)
	log.Info("zipkinURL:", zipkinURL)

	httpAddr := net.JoinHostPort(listenIP, listenPort)
	serviceAddr := net.JoinHostPort(advertisedIP, listenPort)

	collector, err := zipkin.NewHTTPCollector(zipkinURL)
	if err != nil {
		log.Fatal(err)
	}

	recorder := zipkin.NewRecorder(collector, true, serviceAddr, "backend")
	tracer, err = zipkin.NewTracer(recorder,
		zipkin.ClientServerSameSpan(true),
		zipkin.TraceID128Bit(true))
	if err != nil {
		log.Fatal("unable to create Zipkin tracer: ", err)
	}
	opentracing.InitGlobalTracer(tracer)

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
