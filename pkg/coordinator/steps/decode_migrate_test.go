/*
Copyright 2026 The llm-d Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	reqcommon "github.com/llm-d/llm-d-router/pkg/common/request"

	"github.com/llm-d/llm-d-router/pkg/coordinator/config"
	"github.com/llm-d/llm-d-router/pkg/coordinator/gateway"
	"github.com/llm-d/llm-d-router/pkg/coordinator/pipeline"
)

// migrationRequestContext builds a minimal completions RequestContext for the
// migrating decode path, writing to the given recorder.
func migrationRequestContext(recorder http.ResponseWriter) *pipeline.RequestContext {
	return &pipeline.RequestContext{
		RequestID:        "req-mig",
		OriginalPath:     gateway.PathCompletions,
		Model:            "test-model",
		Stream:           false,
		KVTransferParams: map[string]any{},
		Body:             map[string]any{"model": "test-model", "prompt": "Hello"},
		ResponseWriter:   recorder,
	}
}

// TestDecodeStep_ConnectFailureRetriesThenSucceeds drives the migrating decode
// path against a gateway that returns a retryable 503 for the first two
// attempts and then a 200. The client must see only the successful response,
// and each retry must carry the accumulated exclude-endpoints header naming the
// pods that already failed.
func TestDecodeStep_ConnectFailureRetriesThenSucceeds(t *testing.T) {
	var mu sync.Mutex
	var excludeHeaders []string
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		n := attempts
		excludeHeaders = append(excludeHeaders, r.Header.Get(reqcommon.HeaderExcludeEndpoints))
		mu.Unlock()

		w.Header().Set(reqcommon.HeaderDestinationEndpointServed, fmt.Sprintf("10.0.0.%d:8080", n))
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("unavailable"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"text": "ok"}}})
	}))
	defer server.Close()

	gwClient := gateway.New(config.GatewayConfig{Address: server.URL})
	step, err := NewDecodeStep(gwClient, map[string]any{
		ParamEnableRequestMigration: true,
		ParamMaxConnectFailures:     3,
	})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	if err := step.Execute(context.Background(), migrationRequestContext(recorder)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := recorder.Result()
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after retries, got %d", result.StatusCode)
	}
	respBody, _ := io.ReadAll(result.Body)
	if !strings.Contains(string(respBody), "ok") {
		t.Fatalf("expected successful body, got %q", respBody)
	}

	mu.Lock()
	defer mu.Unlock()
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	want := []string{"", "10.0.0.1:8080", "10.0.0.1:8080,10.0.0.2:8080"}
	for i, w := range want {
		if excludeHeaders[i] != w {
			t.Errorf("attempt %d exclude header = %q, want %q", i, excludeHeaders[i], w)
		}
	}
}

// TestDecodeStep_ConnectFailureExhaustedForwardsError verifies that when the
// connect-failure budget is spent the last retryable response is forwarded to
// the client rather than retried forever.
func TestDecodeStep_ConnectFailureExhaustedForwardsError(t *testing.T) {
	attempts := 0
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		attempts++
		mu.Unlock()
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("still unavailable"))
	}))
	defer server.Close()

	gwClient := gateway.New(config.GatewayConfig{Address: server.URL})
	step, err := NewDecodeStep(gwClient, map[string]any{
		ParamEnableRequestMigration: true,
		ParamMaxConnectFailures:     1,
	})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	if err := step.Execute(context.Background(), migrationRequestContext(recorder)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := recorder.Result()
	if result.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 forwarded, got %d", result.StatusCode)
	}
	respBody, _ := io.ReadAll(result.Body)
	if !strings.Contains(string(respBody), "still unavailable") {
		t.Fatalf("expected upstream body forwarded, got %q", respBody)
	}

	mu.Lock()
	defer mu.Unlock()
	if attempts != 2 {
		t.Fatalf("expected 2 attempts (initial + 1 retry), got %d", attempts)
	}
}

// TestDecodeStep_TransportErrorNoRetriesReturns502 points the migrating decode
// path at a closed listener so the round trip fails with a transport error and
// no retries remain, which must surface as a 502 to the client.
func TestDecodeStep_TransportErrorNoRetriesReturns502(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	addr := server.URL
	server.Close() // nothing is listening now, so connections are refused

	gwClient := gateway.New(config.GatewayConfig{Address: addr})
	step, err := NewDecodeStep(gwClient, map[string]any{
		ParamEnableRequestMigration: true,
		ParamMaxConnectFailures:     0,
	})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	if err := step.Execute(context.Background(), migrationRequestContext(recorder)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if recorder.Result().StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502 on transport error, got %d", recorder.Result().StatusCode)
	}
}

// TestDecodeStep_MigrationDisabledDoesNotRetry confirms the default path is
// unchanged: with migration off a single retryable response is forwarded as-is.
func TestDecodeStep_MigrationDisabledDoesNotRetry(t *testing.T) {
	attempts := 0
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		attempts++
		mu.Unlock()
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("unavailable"))
	}))
	defer server.Close()

	gwClient := gateway.New(config.GatewayConfig{Address: server.URL})
	// No migration params: default single-attempt reverse-proxy path.
	step, err := NewDecodeStep(gwClient, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	if err := step.Execute(context.Background(), migrationRequestContext(recorder)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if recorder.Result().StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 forwarded, got %d", recorder.Result().StatusCode)
	}
	mu.Lock()
	defer mu.Unlock()
	if attempts != 1 {
		t.Fatalf("expected exactly 1 attempt with migration disabled, got %d", attempts)
	}
}
