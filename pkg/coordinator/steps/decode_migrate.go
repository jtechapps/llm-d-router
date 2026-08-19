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
	"io"
	"net/http"
	"strings"

	"github.com/go-logr/logr"

	logutil "github.com/llm-d/llm-d-router/pkg/common/observability/logging"
	reqcommon "github.com/llm-d/llm-d-router/pkg/common/request"

	"github.com/llm-d/llm-d-router/pkg/coordinator/pipeline"
)

// hopByHopResponseHeaders are dropped when copying an upstream response head to
// the client; they describe a single transport hop, not the end-to-end message.
var hopByHopResponseHeaders = map[string]bool{
	"connection":          true,
	"keep-alive":          true,
	"proxy-authenticate":  true,
	"proxy-authorization": true,
	"te":                  true,
	"trailer":             true,
	"transfer-encoding":   true,
	"upgrade":             true,
}

// executeMigrating runs the decode call with active request migration enabled.
// Mid-stream migration (resuming a dropped SSE stream on another pod) applies
// only to a bounded slice of requests; everything else gets connect-failure
// retries alone.
func (s *DecodeStep) executeMigrating(ctx context.Context, logger logr.Logger, reqCtx *pipeline.RequestContext) error {
	if s.midStreamEligible(reqCtx) {
		return s.executeMidStreamMigrating(ctx, logger, reqCtx)
	}
	return s.executeConnectRetrying(ctx, logger, reqCtx)
}

// executeConnectRetrying round-trips the decode request itself so a failure that
// occurs before any byte reaches the client can be retried on another pod
// instead of surfacing as a 502. A failure after the response head is on the
// wire cannot be recovered here; that is what mid-stream migration handles.
func (s *DecodeStep) executeConnectRetrying(ctx context.Context, logger logr.Logger, reqCtx *pipeline.RequestContext) error {
	excluded := &excludedEndpoints{}
	connectFailuresLeft := s.migration.maxConnectFailures
	transport := s.gwClient.Transport()
	w := reqCtx.ResponseWriter

	for attempt := 0; ; attempt++ {
		proxyReq, err := newDecodeProxyRequest(ctx, logger, DecodeStepName, reqCtx, s.gwClient, reqCtx.Body, excluded.header())
		if err != nil {
			return err
		}

		resp, err := transport.RoundTrip(proxyReq)
		if err != nil {
			// Connect failure: no response arrived, so nothing has been written to
			// the client and the attempt is safe to retry. The failed pod is
			// unknown here (no response to read the served endpoint from), so the
			// retry is unconstrained.
			if connectFailuresLeft > 0 {
				connectFailuresLeft--
				logger.V(logutil.DEFAULT).Info("decode connect failure, retrying",
					"attempt", attempt, "connectFailuresLeft", connectFailuresLeft, "error", err.Error())
				continue
			}
			logger.Error(err, "decode connect failure, no retries left")
			w.WriteHeader(http.StatusBadGateway)
			return nil
		}

		if isRetryableStatus(resp.StatusCode) && connectFailuresLeft > 0 {
			served := servedEndpoint(resp.Header)
			drainClose(resp.Body)
			connectFailuresLeft--
			excluded.add(served)
			logger.V(logutil.DEFAULT).Info("decode retryable status, retrying",
				"attempt", attempt, "status", resp.StatusCode, "connectFailuresLeft", connectFailuresLeft, "excluded", served)
			continue
		}

		// Terminal response: forward its head and stream its body to the client.
		writeResponseHead(w, resp)
		streamErr := streamDecodeResponse(w, resp.Body)
		resp.Body.Close()
		if streamErr != nil {
			// A read failure after the head is on the wire cannot become a 502; the
			// client sees a truncated response. Requests eligible for mid-stream
			// migration take executeMidStreamMigrating instead of this path.
			logger.Error(streamErr, "decode proxy streaming error: client received a partial response")
		}
		return nil
	}
}

// streamDecodeResponse copies an upstream response body to the client, flushing
// after each chunk so streamed (SSE) output is delivered promptly. It returns
// the first read or write error, or nil on a clean EOF.
func streamDecodeResponse(w http.ResponseWriter, body io.Reader) error {
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		n, readErr := body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

// writeResponseHead copies the upstream response headers (minus hop-by-hop ones)
// and status to the client writer.
func writeResponseHead(w http.ResponseWriter, resp *http.Response) {
	dst := w.Header()
	for key, vals := range resp.Header {
		if hopByHopResponseHeaders[strings.ToLower(key)] {
			continue
		}
		for _, v := range vals {
			dst.Add(key, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
}

// isRetryableStatus reports whether an upstream status is a server-side failure
// worth retrying on another pod. Client errors (4xx) are the request's own
// fault and are forwarded as-is.
func isRetryableStatus(status int) bool {
	return status >= http.StatusInternalServerError
}

// servedEndpoint reads the endpoint that served (or was selected for) a request
// from a response, used to exclude a failed pod on retry. Empty when absent.
func servedEndpoint(h http.Header) string {
	return strings.TrimSpace(h.Get(reqcommon.HeaderDestinationEndpointServed))
}

// drainClose discards a bounded prefix of a response body and closes it, so a
// retried attempt can reuse the transport connection.
func drainClose(rc io.ReadCloser) {
	_, _ = io.Copy(io.Discard, io.LimitReader(rc, maxErrorBodySize))
	_ = rc.Close()
}

// excludedEndpoints accumulates the "<address>:<port>" endpoints a request has
// failed on, deduplicated and in first-seen order, for the exclude-endpoints
// header the EPP reads.
type excludedEndpoints struct {
	set  []string
	seen map[string]bool
}

func (e *excludedEndpoints) add(endpoint string) {
	if endpoint == "" {
		return
	}
	if e.seen == nil {
		e.seen = map[string]bool{}
	}
	if e.seen[endpoint] {
		return
	}
	e.seen[endpoint] = true
	e.set = append(e.set, endpoint)
}

// header returns the exclude-endpoints header for the current exclusion set, or
// nil when nothing is excluded (so no header is stamped).
func (e *excludedEndpoints) header() map[string]string {
	if len(e.set) == 0 {
		return nil
	}
	return map[string]string{reqcommon.HeaderExcludeEndpoints: strings.Join(e.set, ",")}
}
