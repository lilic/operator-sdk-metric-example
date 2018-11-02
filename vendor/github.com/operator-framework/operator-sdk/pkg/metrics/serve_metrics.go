package metrics

import (
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	kcoll "k8s.io/kube-state-metrics/pkg/collectors"
)

type MetricHandler struct {
	c []*kcoll.Collector
}

func ServeMetrics(collectors []*kcoll.Collector) {
	listenAddress := net.JoinHostPort("0.0.0.0", "8080")
	mux := http.NewServeMux()
	mux.Handle("/metrics", &MetricHandler{collectors})
	mux.HandleFunc("/healtz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	// Add index
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Kube Metrics Server</title></head>
             <body>
             <h1>Kube Metrics</h1>
			 <ul>
             <li><a href='` + "/metrics" + `'>metrics</a></li>
             <li><a href='` + "/healthz" + `'>healthz</a></li>
			 </ul>
             </body>
             </html>`))
	})

	fmt.Println(http.ListenAndServe(listenAddress, mux))
}

func (m *MetricHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resHeader := w.Header()
	var writer io.Writer = w

	resHeader.Set("Content-Type", `text/plain; version=`+"0.0.4")

	// Gzip response if requested. Taken from
	// github.com/prometheus/client_golang/prometheus/promhttp.decorateWriter.
	reqHeader := r.Header.Get("Accept-Encoding")
	parts := strings.Split(reqHeader, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "gzip" || strings.HasPrefix(part, "gzip;") {
			writer = gzip.NewWriter(writer)
			resHeader.Set("Content-Encoding", "gzip")
		}
	}

	for _, c := range m.c {
		for _, m := range c.Collect() {
			_, err := fmt.Fprint(writer, *m)
			if err != nil {
				// TODO: Handle panic
				panic(err)
			}
		}
	}

	// In case we gziped the response, we have to close the writer.
	if closer, ok := writer.(io.Closer); ok {
		closer.Close()
	}
}
