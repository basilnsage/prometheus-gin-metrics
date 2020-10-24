package prometrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Registry struct {
	reg prometheus.Registerer
	gat prometheus.Gatherer
}

func NewRegistry() *Registry {
	r := prometheus.NewRegistry()
	r.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	r.MustRegister(prometheus.NewGoCollector())
	return &Registry{
		reg: r,
		gat: r,
	}
}

func (r *Registry) ReportDuration() func(*gin.Context) {
	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "request_duration_seconds",
			Help: "Display duration by code, method, and route",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"code", "method", "route"},
	)
	r.reg.MustRegister(duration)

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		delta := time.Now().Sub(start)

		// create labels
		labels := prometheus.Labels{}
		labels["code"] = strconv.Itoa(c.Writer.Status())
		labels["method"] = c.Request.Method
		labels["route"] = c.FullPath()
		duration.With(labels).Observe(delta.Seconds())
	}
}

func (r *Registry) ReportConcurrentReq() func(*gin.Context) {
	reqStartCount := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "request_start",
		Help: "Number of requests received",
	})
	reqDoneCount := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "request_done",
		Help: "Number of requests completed",
	})
	r.reg.MustRegister(reqStartCount)
	r.reg.MustRegister(reqDoneCount)

	return func(c *gin.Context) {
		reqStartCount.Inc()
		defer reqDoneCount.Inc()
		c.Next()
	}
}

func (r *Registry) DefaultHandler(c *gin.Context) {
	handler := promhttp.InstrumentMetricHandler(
		r.reg, promhttp.HandlerFor(r.gat, promhttp.HandlerOpts{}))
	handler.ServeHTTP(c.Writer, c.Request)
}
