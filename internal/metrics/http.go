package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"strconv"
)

var Requests = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "stockledger_http_requests_total", Help: "HTTP requests handled."}, []string{"method", "status"})

func init() { prometheus.MustRegister(Requests) }

func Register(r prometheus.Registerer) { r.MustRegister(Requests) }

type recorder struct {
	http.ResponseWriter
	status int
}

func (x *recorder) WriteHeader(n int) { x.status = n; x.ResponseWriter.WriteHeader(n) }
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		x := &recorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(x, r)
		Requests.WithLabelValues(r.Method, strconv.Itoa(x.status)).Inc()
	})
}
func Handler() http.Handler { return promhttp.Handler() }
