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
	"bufio"
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

// streamingRequestContext builds a streaming text-prompt completions request,
// the shape eligible for mid-stream migration.
func streamingRequestContext(recorder http.ResponseWriter, maxTokens int) *pipeline.RequestContext {
	return &pipeline.RequestContext{
		RequestID:        "req-stream-mig",
		OriginalPath:     gateway.PathCompletions,
		Model:            "test-model",
		Stream:           true,
		KVTransferParams: map[string]any{},
		Body: map[string]any{
			"model":      "test-model",
			"prompt":     "What is a dachshund? ",
			"max_tokens": maxTokens,
			"stream":     true,
		},
		ResponseWriter: recorder,
	}
}

// writeSSEChunk writes one completions SSE data event carrying a text delta.
func writeSSEChunk(w http.ResponseWriter, id, text string) {
	chunk := map[string]any{
		"id":     id,
		"object": "text_completion",
		"choices": []map[string]any{
			{"index": 0, "text": text, "finish_reason": nil},
		},
		"usage": nil,
	}
	b, _ := json.Marshal(chunk)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// sseChunkIDs parses a forwarded SSE body and returns, per data chunk, its id and
// concatenated text, plus whether a [DONE] terminator was seen.
func sseChunkIDs(t *testing.T, body string) (ids []string, text string, done bool) {
	t.Helper()
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			done = true
			continue
		}
		var chunk map[string]any
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			t.Fatalf("bad chunk %q: %v", payload, err)
		}
		id, _ := chunk["id"].(string)
		ids = append(ids, id)
		text += firstChoiceText(chunk)
	}
	return ids, text, done
}

// TestDecodeStep_MidStreamDropResumesOnAnotherPod drives a decode stream that
// drops after three tokens (no [DONE]) and verifies the coordinator resumes it on
// another pod: the client sees one continuous stream (all four tokens, a single
// id, and a final [DONE]), the resumed request carries the extended prompt and
// reduced max_tokens, and the failed pod is excluded.
func TestDecodeStep_MidStreamDropResumesOnAnotherPod(t *testing.T) {
	var mu sync.Mutex
	var bodies []map[string]any
	var excludeHeaders []string
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		n := attempts
		bodyBytes, _ := io.ReadAll(r.Body)
		var parsed map[string]any
		_ = json.Unmarshal(bodyBytes, &parsed)
		bodies = append(bodies, parsed)
		excludeHeaders = append(excludeHeaders, r.Header.Get(reqcommon.HeaderExcludeEndpoints))
		mu.Unlock()

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set(reqcommon.HeaderDestinationEndpointServed, fmt.Sprintf("10.0.0.%d:8080", n))
		w.WriteHeader(http.StatusOK)

		if n == 1 {
			// Stream three tokens, then drop without a [DONE] terminator.
			writeSSEChunk(w, "cmpl-A", "D")
			writeSSEChunk(w, "cmpl-A", "ach")
			writeSSEChunk(w, "cmpl-A", "sh")
			return
		}
		// Continuation on the second pod, with its own id, ending cleanly.
		writeSSEChunk(w, "cmpl-B", "unds")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	gwClient := gateway.New(config.GatewayConfig{Address: server.URL})
	step, err := NewDecodeStep(gwClient, map[string]any{
		ParamEnableRequestMigration: true,
		ParamMaxConnectFailures:     0,
		ParamMaxRequestMigrations:   2,
	})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	if err := step.Execute(context.Background(), streamingRequestContext(recorder, 25)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ids, text, done := sseChunkIDs(t, recorder.Body.String())
	if text != "Dachshunds" {
		t.Fatalf("client text = %q, want %q", text, "Dachshunds")
	}
	if !done {
		t.Fatal("client stream missing [DONE] terminator")
	}
	for i, id := range ids {
		if id != "cmpl-A" {
			t.Errorf("chunk %d id = %q, want the first id cmpl-A", i, id)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if got := bodies[1]["prompt"]; got != "What is a dachshund? Dachsh" {
		t.Errorf("resumed prompt = %q, want extended prompt", got)
	}
	if got := bodies[1]["max_tokens"]; got != float64(22) {
		t.Errorf("resumed max_tokens = %v, want 22", got)
	}
	if excludeHeaders[0] != "" {
		t.Errorf("first attempt should exclude nothing, got %q", excludeHeaders[0])
	}
	if excludeHeaders[1] != "10.0.0.1:8080" {
		t.Errorf("resume should exclude the failed pod, got %q", excludeHeaders[1])
	}
}

// TestDecodeStep_MidStreamDropExhaustsMigrations verifies that once the migration
// budget is spent a further drop leaves the client with the partial response
// (no [DONE]) rather than looping forever.
func TestDecodeStep_MidStreamDropExhaustsMigrations(t *testing.T) {
	attempts := 0
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		attempts++
		n := attempts
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set(reqcommon.HeaderDestinationEndpointServed, fmt.Sprintf("10.0.0.%d:8080", n))
		w.WriteHeader(http.StatusOK)
		writeSSEChunk(w, "cmpl-A", "x")
		// Always drop without [DONE].
	}))
	defer server.Close()

	gwClient := gateway.New(config.GatewayConfig{Address: server.URL})
	step, err := NewDecodeStep(gwClient, map[string]any{
		ParamEnableRequestMigration: true,
		ParamMaxRequestMigrations:   1,
	})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	if err := step.Execute(context.Background(), streamingRequestContext(recorder, 25)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, done := sseChunkIDs(t, recorder.Body.String())
	if done {
		t.Fatal("client should not see [DONE] when migrations are exhausted")
	}
	mu.Lock()
	defer mu.Unlock()
	// initial attempt + 1 migration = 2 attempts, then the budget is spent.
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

// TestDecodeStep_MidStreamTokenBudgetBlocksMigration verifies a drop is not
// resumed when the generated tokens exceed maxMigratableTokens.
func TestDecodeStep_MidStreamTokenBudgetBlocksMigration(t *testing.T) {
	attempts := 0
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		attempts++
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		writeSSEChunk(w, "cmpl-A", "a")
		writeSSEChunk(w, "cmpl-A", "b")
		// Two tokens produced, cap is 1: drop must not migrate.
	}))
	defer server.Close()

	gwClient := gateway.New(config.GatewayConfig{Address: server.URL})
	step, err := NewDecodeStep(gwClient, map[string]any{
		ParamEnableRequestMigration: true,
		ParamMaxRequestMigrations:   2,
		ParamMaxMigratableTokens:    1,
	})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	if err := step.Execute(context.Background(), streamingRequestContext(recorder, 25)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if attempts != 1 {
		t.Fatalf("expected 1 attempt (no migration past the token cap), got %d", attempts)
	}
}

// TestDecodeStep_MultipleChoicesNotEligibleForMidStream confirms an n>1 request
// is not resumed on a drop: it takes the connect-retry path, so the drop reaches
// the client as a partial response with the upstream called once.
func TestDecodeStep_MultipleChoicesNotEligibleForMidStream(t *testing.T) {
	attempts := 0
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		attempts++
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		writeSSEChunk(w, "cmpl-A", "x")
		// Drop without [DONE].
	}))
	defer server.Close()

	gwClient := gateway.New(config.GatewayConfig{Address: server.URL})
	step, err := NewDecodeStep(gwClient, map[string]any{
		ParamEnableRequestMigration: true,
		ParamMaxRequestMigrations:   2,
	})
	if err != nil {
		t.Fatal(err)
	}

	reqCtx := streamingRequestContext(httptest.NewRecorder(), 25)
	reqCtx.Body["n"] = 2

	if err := step.Execute(context.Background(), reqCtx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if attempts != 1 {
		t.Fatalf("expected 1 attempt (n>1 is not migrated), got %d", attempts)
	}
}

// TestDecodeStep_CleanStreamNoMigration verifies a stream that finishes with
// [DONE] is forwarded verbatim and never triggers a migration.
func TestDecodeStep_CleanStreamNoMigration(t *testing.T) {
	attempts := 0
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		attempts++
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		writeSSEChunk(w, "cmpl-A", "hello")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	gwClient := gateway.New(config.GatewayConfig{Address: server.URL})
	step, err := NewDecodeStep(gwClient, map[string]any{
		ParamEnableRequestMigration: true,
		ParamMaxRequestMigrations:   2,
	})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	if err := step.Execute(context.Background(), streamingRequestContext(recorder, 25)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ids, text, done := sseChunkIDs(t, recorder.Body.String())
	if text != "hello" || !done {
		t.Fatalf("client text = %q done = %v, want %q true", text, done, "hello")
	}
	if len(ids) != 1 || ids[0] != "cmpl-A" {
		t.Fatalf("ids = %v, want [cmpl-A]", ids)
	}
	mu.Lock()
	defer mu.Unlock()
	if attempts != 1 {
		t.Fatalf("expected 1 attempt on a clean stream, got %d", attempts)
	}
}
