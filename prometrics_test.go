package prometrics

import (
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func newEngReg() (*gin.Engine, *Registry) {
	r := NewRegistry()
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	eng.Use(gin.Recovery())
	return eng, r
}

func fetchMetrics(t *testing.T, eng *gin.Engine) map[string]string {
	metricReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricWriter := httptest.NewRecorder()
	eng.ServeHTTP(metricWriter, metricReq)

	// parse metrics
	metrics, err := ioutil.ReadAll(metricWriter.Body)
	if err != nil {
		t.Fatalf("could not read metrics response body: %v", err)
	}

	metricMap := make(map[string]string)
	for _, metric := range strings.Split(string(metrics), "\n") {
		if strings.HasPrefix(metric, "#") || len(strings.TrimSpace(metric)) == 0 {
			continue
		}
		keyVal := strings.Split(metric, " ")
		metricMap[keyVal[0]] = keyVal[1]
	}
	return metricMap
}

// TestUserHandlers - test assigning handlers to a gin engine
func TestUseHandlers(t *testing.T) {
	eng, r := newEngReg()

	eng.Use(r.ReportConcurrentReq())
	eng.Use(r.ReportDuration())
	eng.GET("/metrics", r.DefaultHandler)
}

// TestConcurrentRequests - send a set number of requests to a gin server
// and ensure the result from /metrics is as expected
func TestConcurrentRequests(t *testing.T) {
	eng, r := newEngReg()

	eng.Use(r.ReportConcurrentReq())
	eng.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	eng.GET("/metrics", r.DefaultHandler)

	// make a bunch of requests
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	numRequests := 1000
	var wg sync.WaitGroup
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			writer := httptest.NewRecorder()
			eng.ServeHTTP(writer, req)
		}()
	}
	wg.Wait()

	// fetch metrics
	metricMap := fetchMetrics(t, eng)

	// check num request_start and num request_stop
	start, _ := strconv.Atoi(metricMap["request_start"])
	stop, _ := strconv.Atoi(metricMap["request_done"])
	if start != numRequests+1 {
		t.Errorf("bad request_start: %v, want %v", start, numRequests+1)
	}
	// reqDoneCounter increments AFTER the metrics are returned so
	// the counter never accounts for /metrics finishing
	if stop != numRequests {
		t.Errorf("bad request_done: %v, want %v", stop, numRequests+1)
	}
}

func TestDuration(t *testing.T) {
	eng, r := newEngReg()
	eng.Use(r.ReportDuration())
	eng.GET("/good", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	eng.GET("/bad", func(c *gin.Context) {
		c.Status(http.StatusBadRequest)
	})
	eng.GET("/ugly", func(c *gin.Context) {
		c.Status(http.StatusInternalServerError)
	})
	eng.GET("/metrics", r.DefaultHandler)

	// make requests to generate some metric data
	for _, route := range []string{"/good", "/bad", "/ugly"} {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		writer := httptest.NewRecorder()
		eng.ServeHTTP(writer, req)
	}

	// fetch metrics
	metricMap := fetchMetrics(t, eng)

	// check metrics
	goodRouteMetric := `request_duration_seconds_bucket{code="200",method="GET",route="/good",le="+Inf"}`
	if got, want := metricMap[goodRouteMetric], "1"; got != want {
		t.Errorf("paritioning by 200 response code off: %v, want %v", got, want)
	}
	badMetric := `request_duration_seconds_bucket{code="400",method="GET",route="/bad",le="+Inf"}`
	if got, want := metricMap[badMetric], "1"; got != want {
		t.Errorf("paritioningk by 400 response code off: %v, want %v", got, want)
	}
	uglyMetric := `request_duration_seconds_bucket{code="500",method="GET",route="/ugly",le="+Inf"}`
	if got, want := metricMap[uglyMetric], "1"; got != want {
		t.Errorf("partitioning by 500 response code off: %v, want %v", got, want)
	}
}
