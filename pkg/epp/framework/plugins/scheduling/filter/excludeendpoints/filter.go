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

package excludeendpoints

import (
	"context"
	"encoding/json"
	"strings"

	reqcommon "github.com/llm-d/llm-d-router/pkg/common/request"
	"github.com/llm-d/llm-d-router/pkg/epp/framework/interface/plugin"
	"github.com/llm-d/llm-d-router/pkg/epp/framework/interface/scheduling"
)

// ExcludeEndpointsFilterType is the type of the ExcludeEndpoints filter.
const ExcludeEndpointsFilterType = "exclude-endpoints-filter"

var _ scheduling.Filter = &ExcludeEndpoints{} // validate interface conformance

// Factory defines the factory function for the ExcludeEndpoints filter. The
// filter takes no parameters: its exclusion set is supplied per request via the
// exclude-endpoints header.
func Factory(name string, _ *json.Decoder, _ plugin.Handle) (plugin.Plugin, error) {
	return NewExcludeEndpoints(name), nil
}

// NewExcludeEndpoints returns an ExcludeEndpoints filter with the given name.
func NewExcludeEndpoints(name string) *ExcludeEndpoints {
	if name == "" {
		name = ExcludeEndpointsFilterType
	}
	return &ExcludeEndpoints{typedName: plugin.TypedName{Type: ExcludeEndpointsFilterType, Name: name}}
}

// ExcludeEndpoints drops candidate endpoints named in the request's
// exclude-endpoints header, the deny-list complement of the subset allow-list.
// The coordinator sets the header on a migration retry to steer the request off
// a backend that just failed it.
type ExcludeEndpoints struct {
	typedName plugin.TypedName
}

// TypedName returns the typed name of the plugin.
func (f *ExcludeEndpoints) TypedName() plugin.TypedName {
	return f.typedName
}

// WithName sets the name of the plugin.
func (f *ExcludeEndpoints) WithName(name string) *ExcludeEndpoints {
	f.typedName.Name = name
	return f
}

// Filter removes endpoints whose IP address appears in the request's
// exclude-endpoints header. The header is a comma-separated list of
// "<address>:<port>" entries; matching is by address only, aligning with the
// subset allow-list in requestcontrol.DatastoreEndpointCandidates. When the
// header is absent the candidates pass through unchanged. When excluding would
// empty the pool the original candidates are returned, since routing to a
// recently-failed pod (retries are bounded by the coordinator) beats a hard
// scheduling failure.
func (f *ExcludeEndpoints) Filter(_ context.Context, request *scheduling.InferenceRequest, endpoints []scheduling.Endpoint) []scheduling.Endpoint {
	excluded := parseExcluded(request)
	if len(excluded) == 0 {
		return endpoints
	}

	filtered := make([]scheduling.Endpoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if _, drop := excluded[endpoint.GetMetadata().GetIPAddress()]; drop {
			continue
		}
		filtered = append(filtered, endpoint)
	}

	if len(filtered) == 0 {
		return endpoints
	}
	return filtered
}

// parseExcluded reads the exclude-endpoints header and returns the set of IP
// addresses to drop, stripping the port from each "<address>:<port>" entry.
func parseExcluded(request *scheduling.InferenceRequest) map[string]struct{} {
	if request == nil {
		return nil
	}
	raw := strings.TrimSpace(request.Headers[reqcommon.HeaderExcludeEndpoints])
	if raw == "" {
		return nil
	}

	excluded := map[string]struct{}{}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if idx := strings.LastIndexByte(entry, ':'); idx >= 0 {
			entry = entry[:idx]
		}
		excluded[entry] = struct{}{}
	}
	return excluded
}
