package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	//"github.com/prometheus/client_golang/prometheus"
	//"github.com/prometheus/client_golang/prometheus/promhttp"
)

var duration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "request_duration_seconds",
		Help: "Display duration by code, method, and route",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"code", "method", "route"},
)

func reportDuration(c *gin.Context) {
	log.Print("sample request start time")
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

func handler2(c *gin.Context) {
	log.Print("handler2 called")
	c.Next()
	log.Print("handler2: post Next()")
}


func main() {
	// register prometheus reporters
	prometheus.MustRegister(duration)

	eng := gin.Default()
	eng.Use(handler2)
	eng.Use(reportDuration)

	eng.GET("/test", func(c *gin.Context) {
		time.Sleep(time.Duration(rand.Float64() * 2.0 * float64(time.Second)))
		c.String(http.StatusOK, "/test called")
	})
	eng.GET("/metrics", func(c *gin.Context) {
		promHandler := promhttp.Handler()
		promHandler.ServeHTTP(c.Writer, c.Request)
	})

	log.Print("run on localhost:4567")
	if err := eng.Run(":4567"); err != nil {
		panic(err)
	}
}