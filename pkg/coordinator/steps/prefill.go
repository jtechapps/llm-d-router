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
	"errors"
	"fmt"
	"maps"
	"net/http"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	logutil "github.com/llm-d/llm-d-router/pkg/common/observability/logging"
	reqcommon "github.com/llm-d/llm-d-router/pkg/common/request"

	"github.com/llm-d/llm-d-router/pkg/coordinator/common/httplog"
	"github.com/llm-d/llm-d-router/pkg/coordinator/connectors/ec"
	"github.com/llm-d/llm-d-router/pkg/coordinator/connectors/kv"
	"github.com/llm-d/llm-d-router/pkg/coordinator/gateway"
	"github.com/llm-d/llm-d-router/pkg/coordinator/pipeline"
)

const PrefillStepName = "prefill"

func init() {
	pipeline.Register(PrefillStepName, NewPrefillStep)
}

type PrefillStep struct {
	useOpenAIFormat bool
	gwClient        *gateway.Client
	kv              kv.Connector
	ec              ec.Connector
	migration       migrationConfig
}

func NewPrefillStep(gwClient *gateway.Client, params map[string]any) (pipeline.Step, error) {
	if gwClient == nil {
		return nil, errors.New("prefill: gateway client is required")
	}
	useOpenAI, err := parseUseOpenAIFormat(params)
	if err != nil {
		return nil, fmt.Errorf("prefill: %w", err)
	}
	kvName, err := paramString(params, ParamKVConnector)
	if err != nil {
		return nil, fmt.Errorf("prefill: %w", err)
	}
	kvConn, err := kv.Build(kvName)
	if err != nil {
		return nil, fmt.Errorf("prefill: %w", err)
	}
	ecName, err := paramString(params, ParamECConnector)
	if err != nil {
		return nil, fmt.Errorf("prefill: %w", err)
	}
	ecConn, err := ec.Build(ecName)
	if err != nil {
		return nil, fmt.Errorf("prefill: %w", err)
	}
	migration, err := parseMigrationConfig(params)
	if err != nil {
		return nil, fmt.Errorf("prefill: %w", err)
	}
	return &PrefillStep{useOpenAIFormat: useOpenAI, gwClient: gwClient, kv: kvConn, ec: ecConn, migration: migration}, nil
}

func (s *PrefillStep) Name() string { return PrefillStepName }

func (s *PrefillStep) Execute(ctx context.Context, reqCtx *pipeline.RequestContext) error {
	logger := log.FromContext(ctx).WithName(PrefillStepName)

	features := buildMMFeatures(reqCtx.MultimodalEntries, true)

	format := resolveFormat(s.useOpenAIFormat, reqCtx.OriginalPath)
	body, err := s.buildPrefillBody(ctx, reqCtx, features, format)
	if err != nil {
		return fmt.Errorf("prefill: %w", err)
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("prefill: marshal: %w", err)
	}

	path := gateway.PathForFormat(format)
	logger.V(logutil.DEFAULT).Info("sending request", "path", path)

	headers := reqCtx.ForwardedHeaders()
	headers[reqcommon.RequestIDHeaderKey] = reqCtx.RequestID
	headers[gateway.EPPProfileHeader] = gateway.PhasePrefill

	if v := logger.V(logutil.DEBUG); v.Enabled() {
		v.Info("request body", "method", "POST", "path", path, "bodyLen", len(bodyBytes), "headers", httplog.RedactedHeaders(headers))
	}

	resp, err := s.postWithMigration(ctx, logger, path, bodyBytes, headers)
	if err != nil {
		return fmt.Errorf("prefill: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody := readErrorBody(resp.Body)
		return upstreamError(PrefillStepName, resp.StatusCode, respBody)
	}

	var prefillResp prefillResponse
	if err := json.NewDecoder(resp.Body).Decode(&prefillResp); err != nil {
		return fmt.Errorf("prefill: decode response: %w", err)
	}

	reqCtx.KVTransferParams = coerceParamsMap(logger, prefillResp.KVTransferParams, "kv_transfer_params")

	logger.V(logutil.DEFAULT).Info("complete")
	return nil
}

// postWithMigration sends the prefill request, retrying a connect failure or a
// retryable upstream status on another pod when request migration is enabled.
// Prefill produces no client output, so only connect-failure retries apply; the
// caller handles a returned non-retryable or retries-exhausted response as
// before. Excluding the failed pod is best-effort: a pure connect failure
// carries no served endpoint.
func (s *PrefillStep) postWithMigration(ctx context.Context, logger logr.Logger, path string, bodyBytes []byte, headers map[string]string) (*http.Response, error) {
	if !s.migration.enabled {
		return s.gwClient.Post(ctx, path, bodyBytes, headers)
	}

	excluded := &excludedEndpoints{}
	connectFailuresLeft := s.migration.maxConnectFailures
	for attempt := 0; ; attempt++ {
		for k, v := range excluded.header() {
			headers[k] = v
		}

		resp, err := s.gwClient.Post(ctx, path, bodyBytes, headers)
		if err != nil {
			if connectFailuresLeft > 0 {
				connectFailuresLeft--
				logger.V(logutil.DEFAULT).Info("prefill connect failure, retrying",
					"attempt", attempt, "connectFailuresLeft", connectFailuresLeft, "error", err.Error())
				continue
			}
			return nil, err
		}

		if isRetryableStatus(resp.StatusCode) && connectFailuresLeft > 0 {
			served := servedEndpoint(resp.Header)
			drainClose(resp.Body)
			connectFailuresLeft--
			excluded.add(served)
			logger.V(logutil.DEFAULT).Info("prefill retryable status, retrying",
				"attempt", attempt, "status", resp.StatusCode, "connectFailuresLeft", connectFailuresLeft, "excluded", served)
			continue
		}

		return resp, nil
	}
}

func (s *PrefillStep) buildPrefillBody(ctx context.Context, reqCtx *pipeline.RequestContext, features map[string]any, format gateway.RequestFormat) (map[string]any, error) {
	ecParams, err := s.ec.PreparePrefillECParams(ctx, reqCtx)
	if err != nil {
		return nil, err
	}
	kvParams := s.kv.PreparePrefillKVParams(ctx, reqCtx)

	switch format {
	case gateway.FormatChatCompletions:
		body := maps.Clone(reqCtx.Body)
		capSingleTokenOutput(body, format)
		tokens := map[string]any{
			"token_ids": reqCtx.TokenIDs,
		}
		if features != nil {
			tokensFeatures := map[string]any{
				"mm_hashes":       features["mm_hashes"],
				"mm_placeholders": features["mm_placeholders"],
			}
			tokens["features"] = tokensFeatures
		}
		body["tokens"] = tokens
		body[reqcommon.FieldKVTransferParams] = kvParams
		if len(ecParams) > 0 {
			body[reqcommon.FieldECTransferParams] = ecParams
		}
		return body, nil

	case gateway.FormatCompletions:
		prompt := reqCtx.Body["prompt"]
		if len(reqCtx.TokenIDs) > 0 {
			prompt = reqCtx.TokenIDs
		}
		body := map[string]any{
			"request_id":                    reqCtx.RequestID,
			"model":                         reqCtx.Model,
			"prompt":                        prompt,
			reqcommon.FieldKVTransferParams: kvParams,
		}
		capSingleTokenOutput(body, format)
		if features != nil {
			body["features"] = features
		}
		if len(ecParams) > 0 {
			body[reqcommon.FieldECTransferParams] = ecParams
		}
		return body, nil

	case gateway.FormatGenerate:
		// The /inference/v1/generate engine reads transfer params only from
		// sampling_params.extra_args; top-level fields are ignored on input.
		sampling := map[string]any{reqcommon.FieldMaxTokens: 1}
		setGenerateTransferParams(sampling, kvParams, ecParams)
		body := map[string]any{
			"request_id":                  reqCtx.RequestID,
			"token_ids":                   reqCtx.TokenIDs,
			"model":                       reqCtx.Model,
			reqcommon.FieldSamplingParams: sampling,
		}
		capSingleTokenOutput(body, format)
		if features != nil {
			body["features"] = features
		}
		return body, nil
	}
	// resolveFormat only ever yields the three formats above; a new value
	// reaching here is a programming error, not a client fault.
	return nil, fmt.Errorf("prefill: unsupported request format %v", format)
}

type prefillResponse struct {
	// KVTransferParams is decoded as any (not map[string]any) so a non-object
	// value does not fail the decode; coerceParamsMap coerces it.
	KVTransferParams any `json:"kv_transfer_params"`
}
