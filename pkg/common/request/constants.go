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

package request

const (
	RequestIDHeaderKey = "x-request-id"

	// HeaderExcludeEndpoints is set by the coordinator on a migration retry to
	// tell the EPP not to schedule the request onto the listed endpoints. The
	// value is a comma-separated list of "<address>:<port>" entries, matching the
	// endpoint-subset filter's format. Consumed by the EPP exclude-endpoints
	// scheduling filter.
	HeaderExcludeEndpoints = "x-gateway-destination-endpoint-exclude"

	// HeaderDestinationEndpointServed is the response header/metadata key Envoy
	// uses to report the endpoint that served (or was selected for) a request.
	// The coordinator reads it from a failed attempt to name the pod to exclude
	// on retry; it is best-effort, since a pure connect failure carries no
	// response and therefore no served endpoint.
	HeaderDestinationEndpointServed = "x-gateway-destination-endpoint-served"

	FieldKVTransferParams     = "kv_transfer_params"
	FieldECTransferParams     = "ec_transfer_params"
	FieldMaxOutputTokens      = "max_output_tokens" // Used by Responses API
	FieldMinTokens            = "min_tokens"
	FieldSamplingParams       = "sampling_params"
	FieldExtraArgs            = "extra_args"
	FieldDoRemotePrefill      = "do_remote_prefill"
	FieldDoRemoteDecode       = "do_remote_decode"
	FieldRemoteBlockIDs       = "remote_block_ids"
	FieldRemoteEngineID       = "remote_engine_id"
	FieldRemoteHost           = "remote_host"
	FieldRemotePort           = "remote_port"
	FieldCacheHitThreshold    = "cache_hit_threshold"
	FieldContinueFinalMessage = "continue_final_message"
	FieldAddGenerationPrompt  = "add_generation_prompt"
)
