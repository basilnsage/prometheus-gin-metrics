package prometrics

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestUserHandlers - test assigning handlers to a gin engine
func  TestUseHandlers(t *testing.T) {
	r := NewRegistry()
	gin.SetMode(gin.TestMode)
	eng := gin.Default()

	eng.Use(r.ReportConcurrentReq())
	eng.Use(r.ReportDuration())
	eng.GET("/metrics", r.DefaultHandler)
}

// TestConcurrentRequests - send a set number of requests to a gin server
// and ensure the result from /metrics is as expected
func TestConcurrentRequests(t *testing.T) {
	r := NewRegistry()
	gin.SetMode(gin.TestMode)
	eng := gin.Default()

	eng.Use(r.ReportConcurrentReq())
	eng.GET("/test", func (c *gin.Context) {
		c.Status(http.StatusOK)
	})
	eng.GET("/metrics", r.DefaultHandler)

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

	metricReq  := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricWriter := httptest.NewRecorder()
	eng.ServeHTTP(metricWriter, metricReq)

	metrics, err := ioutil.ReadAll(metricWriter.Body)
	if err != nil {
		t.Errorf("ioutil.Readall: %v", err)
	}

	metricMap := make(map[string]string)
	for _, metric := range strings.Split(string(metrics), "\n") {
		if strings.HasPrefix(metric, "#") || len(strings.TrimSpace(metric)) == 0 {
			continue
		}
		keyVal := strings.Split(metric, " ")
		metricMap[keyVal[0]] = keyVal[1]
	}

	time.Sleep(10 * time.Second)

	start, _ := strconv.Atoi(metricMap["request_start"])
	stop, _ := strconv.Atoi(metricMap["request_done"])
	if start != numRequests + 1 {
		t.Errorf("bad request_start: %v, want %v", start, numRequests + 1)
	}
	if stop != numRequests + 1 {
		t.Errorf("bad request_done: %v, want %v", stop, numRequests + 1)
	}
}