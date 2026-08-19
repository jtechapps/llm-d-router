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
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-logr/logr"

	logutil "github.com/llm-d/llm-d-router/pkg/common/observability/logging"
	reqcommon "github.com/llm-d/llm-d-router/pkg/common/request"

	"github.com/llm-d/llm-d-router/pkg/coordinator/gateway"
	"github.com/llm-d/llm-d-router/pkg/coordinator/pipeline"
)

// sseDataPrefix marks a server-sent-event data line; sseDone is the OpenAI
// stream terminator carried on such a line.
const (
	sseDataPrefix = "data:"
	sseDone       = "[DONE]"
)

// midStreamEligible reports whether a request can be resumed after a mid-stream
// decode drop. Resumption appends the generated text to the prompt and continues
// on another pod, which is only well defined for a streaming text-prompt
// /v1/completions request producing a single choice. Everything else (chat
// completions, tokenized prompts, n > 1, non-streaming) falls back to
// connect-failure retries alone.
func (s *DecodeStep) midStreamEligible(reqCtx *pipeline.RequestContext) bool {
	if s.migration.maxRequestMigrations <= 0 || !reqCtx.Stream {
		return false
	}
	if resolveFormat(s.useOpenAIFormat, reqCtx.OriginalPath) != gateway.FormatCompletions {
		return false
	}
	if _, ok := reqCtx.Body["prompt"].(string); !ok {
		return false
	}
	if n, ok := reqCtx.Body["n"]; ok && !isOne(n) {
		return false
	}
	return true
}

// executeMidStreamMigrating streams a decode response and, if the stream drops
// before the [DONE] terminator, resumes it on another pod: the generated text so
// far is appended to the prompt, max_tokens is reduced by the number of tokens
// already produced, and the failed pod is excluded. The client sees one
// uninterrupted stream because resumed chunks are rewritten to carry the id of
// the first chunk it received.
//
// A failure before the first byte reaches the client is a connect failure and
// uses the maxConnectFailures budget; a failure after the client has seen output
// uses the maxRequestMigrations budget. Once either budget is spent the client
// keeps whatever it has received so far.
func (s *DecodeStep) executeMidStreamMigrating(ctx context.Context, logger logr.Logger, reqCtx *pipeline.RequestContext) error {
	excluded := &excludedEndpoints{}
	connectFailuresLeft := s.migration.maxConnectFailures
	migrationsLeft := s.migration.maxRequestMigrations
	transport := s.gwClient.Transport()
	w := reqCtx.ResponseWriter

	stream := &completionsStream{}
	headWritten := false

	for {
		proxyReq, err := newDecodeProxyRequest(ctx, logger, DecodeStepName, reqCtx, s.gwClient, reqCtx.Body, excluded.header())
		if err != nil {
			return err
		}

		resp, err := transport.RoundTrip(proxyReq)
		if err != nil {
			if !headWritten {
				if connectFailuresLeft > 0 {
					connectFailuresLeft--
					logger.V(logutil.DEFAULT).Info("decode connect failure, retrying",
						"connectFailuresLeft", connectFailuresLeft, "error", err.Error())
					continue
				}
				logger.Error(err, "decode connect failure, no retries left")
				w.WriteHeader(http.StatusBadGateway)
				return nil
			}
			// The client already holds part of the stream; a failed re-issue spends
			// a migration and tries again on another pod.
			if migrationsLeft > 0 {
				migrationsLeft--
				logger.V(logutil.DEFAULT).Info("decode re-issue connect failure, retrying",
					"migrationsLeft", migrationsLeft, "error", err.Error())
				continue
			}
			logger.Error(err, "decode re-issue connect failure, no migrations left: client received a partial response")
			return nil
		}

		if isRetryableStatus(resp.StatusCode) {
			served := servedEndpoint(resp.Header)
			drainClose(resp.Body)
			if !headWritten {
				if connectFailuresLeft > 0 {
					connectFailuresLeft--
					excluded.add(served)
					logger.V(logutil.DEFAULT).Info("decode retryable status, retrying",
						"status", resp.StatusCode, "connectFailuresLeft", connectFailuresLeft, "excluded", served)
					continue
				}
				// Budget spent before any output: forward the error to the client.
				writeResponseHead(w, resp)
				return nil
			}
			if migrationsLeft > 0 {
				migrationsLeft--
				excluded.add(served)
				logger.V(logutil.DEFAULT).Info("decode re-issue retryable status, retrying",
					"status", resp.StatusCode, "migrationsLeft", migrationsLeft, "excluded", served)
				continue
			}
			logger.V(logutil.DEFAULT).Info("decode re-issue retryable status, no migrations left: client received a partial response",
				"status", resp.StatusCode)
			return nil
		}

		if !headWritten {
			writeResponseHead(w, resp)
			headWritten = true
		}

		outcome := stream.forward(w, resp.Body)
		served := servedEndpoint(resp.Header)
		resp.Body.Close()

		if outcome == streamComplete {
			return nil
		}

		// The stream dropped before [DONE]. Migrate if the budget and the token cap
		// allow it, otherwise leave the client with the partial response.
		if migrationsLeft <= 0 {
			logger.Error(errors.New("stream dropped"), "decode stream dropped, no migrations left: client received a partial response")
			return nil
		}
		if !stream.withinTokenBudget(s.migration.maxMigratableTokens, len(reqCtx.TokenIDs)) {
			logger.Error(errors.New("stream dropped"), "decode stream dropped, migratable-token cap exceeded: client received a partial response",
				"produced", stream.produced, "maxMigratableTokens", s.migration.maxMigratableTokens)
			return nil
		}

		migrationsLeft--
		excluded.add(served)
		stream.mutateForResume(reqCtx)
		logger.V(logutil.DEFAULT).Info("decode stream dropped, migrating",
			"migrationsLeft", migrationsLeft, "excluded", served, "produced", stream.produced)
	}
}

// streamOutcome reports how a forwarded stream ended.
type streamOutcome int

const (
	// streamComplete means the [DONE] terminator was forwarded.
	streamComplete streamOutcome = iota
	// streamDropped means the body ended (EOF or error) before [DONE].
	streamDropped
)

// completionsStream forwards a /v1/completions SSE stream to the client while
// tracking the state a resumed continuation needs: the id of the first chunk the
// client saw (so resumed chunks can adopt it) and the generated text and token
// count so far (to extend the prompt and reduce max_tokens).
type completionsStream struct {
	firstID  string
	haveID   bool
	resuming bool
	output   strings.Builder
	produced int
}

// forward relays SSE events to the client until the [DONE] terminator or a
// premature end. On the first (non-resuming) pass it forwards each event's bytes
// unchanged; on a resumed pass it rewrites each chunk's id to the first id the
// client saw so the client observes one continuous stream. Either way it
// accumulates the generated text and token count.
func (c *completionsStream) forward(w http.ResponseWriter, body io.Reader) streamOutcome {
	flusher, _ := w.(http.Flusher)
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			payload, isData := ssePayload(line)
			if isData && payload == sseDone {
				// [DONE] arrives only from a stream that ran to completion, so it is
				// always forwarded, whether or not this pass was a resumption.
				_, _ = io.WriteString(w, line)
				if flusher != nil {
					flusher.Flush()
				}
				return streamComplete
			}

			out := line
			if isData {
				if rewritten, ok := c.consume(payload); ok {
					out = sseDataPrefix + " " + rewritten + "\n"
				}
			}
			if _, werr := io.WriteString(w, out); werr != nil {
				return streamDropped
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			// A clean EOF here means the stream ended without [DONE]: a drop.
			return streamDropped
		}
	}
}

// consume parses a completions chunk payload, accumulating its generated text and
// (when resuming) rewriting its id to the first id the client saw. It returns the
// possibly-rewritten payload and whether the payload should be replaced with it.
// A payload it cannot parse is forwarded verbatim.
func (c *completionsStream) consume(payload string) (string, bool) {
	var chunk map[string]any
	if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
		return "", false
	}

	if id, ok := chunk["id"].(string); ok && !c.haveID {
		c.firstID = id
		c.haveID = true
	}

	if text := firstChoiceText(chunk); text != "" {
		c.output.WriteString(text)
		c.produced++
	}

	if !c.resuming || !c.haveID {
		return "", false
	}
	chunk["id"] = c.firstID
	rewritten, err := json.Marshal(chunk)
	if err != nil {
		return "", false
	}
	return string(rewritten), true
}

// withinTokenBudget reports whether resuming stays within the migratable-token
// cap. The cap bounds the prompt tokens plus the tokens generated so far; a cap
// of zero means no limit.
func (c *completionsStream) withinTokenBudget(maxMigratableTokens, promptTokens int) bool {
	if maxMigratableTokens <= 0 {
		return true
	}
	return promptTokens+c.produced <= maxMigratableTokens
}

// mutateForResume extends the request for a continuation on another pod: the
// generated text is appended to the prompt and max_tokens is reduced by the
// number of tokens already produced. Subsequent chunks are rewritten to the
// client-visible id.
func (c *completionsStream) mutateForResume(reqCtx *pipeline.RequestContext) {
	if prompt, ok := reqCtx.Body["prompt"].(string); ok {
		reqCtx.Body["prompt"] = prompt + c.output.String()
	}
	if remaining, ok := reduceMaxTokens(reqCtx.Body, c.produced); ok {
		reqCtx.Body[reqcommon.FieldMaxTokens] = remaining
	}
	c.resuming = true
}

// reduceMaxTokens computes max_tokens minus the tokens already produced, floored
// at 1. It returns false when the body carries no integral max_tokens to reduce.
func reduceMaxTokens(body map[string]any, produced int) (int, bool) {
	current, ok := asInt(body[reqcommon.FieldMaxTokens])
	if !ok {
		return 0, false
	}
	remaining := current - produced
	if remaining < 1 {
		remaining = 1
	}
	return remaining, true
}

// ssePayload extracts the value of an SSE data line, trimming the "data:" prefix
// and surrounding whitespace. It reports false for any other line (blank lines,
// comments, event/id fields).
func ssePayload(line string) (string, bool) {
	trimmed := strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(trimmed, sseDataPrefix) {
		return "", false
	}
	return strings.TrimSpace(trimmed[len(sseDataPrefix):]), true
}

// firstChoiceText returns the text of the first choice in a completions chunk, or
// the empty string when absent.
func firstChoiceText(chunk map[string]any) string {
	choices, ok := chunk["choices"].([]any)
	if !ok || len(choices) == 0 {
		return ""
	}
	choice, ok := choices[0].(map[string]any)
	if !ok {
		return ""
	}
	text, _ := choice["text"].(string)
	return text
}

// isOne reports whether a JSON-decoded number equals 1.
func isOne(v any) bool {
	n, ok := asInt(v)
	return ok && n == 1
}

// asInt coerces a JSON-decoded number (float64 or json.Number) to an int,
// reporting false for non-numeric or non-integral values.
func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		if n != float64(int(n)) {
			return 0, false
		}
		return int(n), true
	case int:
		return n, true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	}
	return 0, false
}
