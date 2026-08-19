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

import "testing"

func TestParseMigrationConfig(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		want    migrationConfig
		wantErr bool
	}{
		{
			name:   "absent yields disabled zero value",
			params: map[string]any{},
			want:   migrationConfig{},
		},
		{
			name: "all fields set",
			params: map[string]any{
				ParamEnableRequestMigration: true,
				ParamMaxConnectFailures:     3,
				ParamMaxRequestMigrations:   2,
				ParamMaxMigratableTokens:    50000,
			},
			want: migrationConfig{
				enabled:              true,
				maxConnectFailures:   3,
				maxRequestMigrations: 2,
				maxMigratableTokens:  50000,
			},
		},
		{
			name: "float-formatted ints are accepted",
			params: map[string]any{
				ParamEnableRequestMigration: true,
				ParamMaxConnectFailures:     float64(3),
			},
			want: migrationConfig{enabled: true, maxConnectFailures: 3},
		},
		{
			name: "bounds may be set while disabled",
			params: map[string]any{
				ParamMaxConnectFailures: 1,
			},
			want: migrationConfig{maxConnectFailures: 1},
		},
		{
			name:    "negative connect failures rejected",
			params:  map[string]any{ParamMaxConnectFailures: -1},
			wantErr: true,
		},
		{
			name:    "negative request migrations rejected",
			params:  map[string]any{ParamMaxRequestMigrations: -2},
			wantErr: true,
		},
		{
			name:    "negative migratable tokens rejected",
			params:  map[string]any{ParamMaxMigratableTokens: -5},
			wantErr: true,
		},
		{
			name:    "non-bool enable rejected",
			params:  map[string]any{ParamEnableRequestMigration: "yes"},
			wantErr: true,
		},
		{
			name:    "non-integer float rejected",
			params:  map[string]any{ParamMaxConnectFailures: 1.5},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMigrationConfig(tt.params)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none (config %+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("parseMigrationConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
