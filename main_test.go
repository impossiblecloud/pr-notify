package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestHealthHandler(t *testing.T) {

	// Create request
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)

	// Run the handler
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := `Health is OK`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestRootHandler(t *testing.T) {

	// Create metrics
	var testMetrics appMetrics
	registry := prometheus.NewRegistry()

	testMetrics = initMetrics(registry, []string{}, []string{})
	registry.MustRegister()

	// Create request
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(endpointCounter(rootHandler, testMetrics))

	// Run the handler
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := fmt.Sprintf("Up and running. Version: %s", version)
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}

	// Check metrics
	if counter, err := getCounter(testMetrics.httpRequestsTotal, testMetrics.labelValues...); err == nil {
		if counter != 1 {
			t.Errorf("wrong httpRequestsTotal counter: %v, expected: 1", counter)
		}
	} else {
		t.Errorf("failed to get counter: %s", err.Error())
	}

}
