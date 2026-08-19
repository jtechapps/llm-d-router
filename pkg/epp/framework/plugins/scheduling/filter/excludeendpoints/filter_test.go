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
	"testing"

	"github.com/stretchr/testify/assert"
	k8stypes "k8s.io/apimachinery/pkg/types"

	reqcommon "github.com/llm-d/llm-d-router/pkg/common/request"
	fwkdl "github.com/llm-d/llm-d-router/pkg/epp/framework/interface/datalayer"
	"github.com/llm-d/llm-d-router/pkg/epp/framework/interface/scheduling"
	"github.com/llm-d/llm-d-router/test/utils"
)

func createEndpoint(name, ipaddr string) scheduling.Endpoint {
	return scheduling.NewEndpoint(
		&fwkdl.EndpointMetadata{
			ID:      k8stypes.NamespacedName{Namespace: "default", Name: name},
			Address: ipaddr,
			Port:    "8080",
		},
		&fwkdl.Metrics{},
		nil,
	)
}

func TestExcludeEndpointsFiltering(t *testing.T) {
	endpoints := []scheduling.Endpoint{
		createEndpoint("pod-1", "10.0.0.1"),
		createEndpoint("pod-2", "10.0.0.2"),
		createEndpoint("pod-3", "10.0.0.3"),
	}

	tests := []struct {
		name         string
		header       string
		headerAbsent bool
		expectedPods []string
	}{
		{
			name:         "no header passes all candidates through",
			headerAbsent: true,
			expectedPods: []string{"pod-1", "pod-2", "pod-3"},
		},
		{
			name:         "empty header passes all candidates through",
			header:       "",
			expectedPods: []string{"pod-1", "pod-2", "pod-3"},
		},
		{
			name:         "single excluded pod is dropped",
			header:       "10.0.0.1:8080",
			expectedPods: []string{"pod-2", "pod-3"},
		},
		{
			name:         "multiple excluded pods are dropped",
			header:       "10.0.0.1:8080,10.0.0.3:8080",
			expectedPods: []string{"pod-2"},
		},
		{
			name:         "address without a port still matches",
			header:       "10.0.0.2",
			expectedPods: []string{"pod-1", "pod-3"},
		},
		{
			name:         "whitespace and empty entries are ignored",
			header:       " 10.0.0.1:8080 , ,10.0.0.2:8080 ",
			expectedPods: []string{"pod-3"},
		},
		{
			name:         "unknown excluded address leaves candidates unchanged",
			header:       "10.9.9.9:8080",
			expectedPods: []string{"pod-1", "pod-2", "pod-3"},
		},
		{
			name:         "excluding every candidate falls back to all candidates",
			header:       "10.0.0.1:8080,10.0.0.2:8080,10.0.0.3:8080",
			expectedPods: []string{"pod-1", "pod-2", "pod-3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if !tt.headerAbsent {
				headers[reqcommon.HeaderExcludeEndpoints] = tt.header
			}
			request := &scheduling.InferenceRequest{Headers: headers}

			filter := NewExcludeEndpoints("test-exclude")
			ctx := utils.NewTestContext(t)

			filtered := filter.Filter(ctx, request, endpoints)

			actual := make([]string, len(filtered))
			for i, ep := range filtered {
				actual[i] = ep.GetMetadata().ID.Name
			}
			assert.ElementsMatch(t, tt.expectedPods, actual)
		})
	}
}

func TestExcludeEndpointsNilRequest(t *testing.T) {
	endpoints := []scheduling.Endpoint{createEndpoint("pod-1", "10.0.0.1")}
	filter := NewExcludeEndpoints("test-exclude")
	filtered := filter.Filter(utils.NewTestContext(t), nil, endpoints)
	assert.Len(t, filtered, 1)
}

func TestFactory(t *testing.T) {
	p, err := Factory("test-exclude", nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, p)

	filter, ok := p.(*ExcludeEndpoints)
	assert.True(t, ok)
	assert.Equal(t, ExcludeEndpointsFilterType, filter.TypedName().Type)
	assert.Equal(t, "test-exclude", filter.TypedName().Name)
}
