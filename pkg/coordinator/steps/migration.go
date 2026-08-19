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

import "fmt"

// Active-request-migration step parameter keys. Snake_case to match the other
// step params (use_openai_format, kv_connector); the design doc's camelCase is
// illustrative only.
const (
	ParamEnableRequestMigration = "enable_request_migration"
	ParamMaxConnectFailures     = "max_connect_failures"
	ParamMaxRequestMigrations   = "max_request_migrations"
	ParamMaxMigratableTokens    = "max_migratable_tokens"
)

// migrationConfig holds a step's active-request-migration settings. The zero
// value is migration disabled, which is the behavior when the step carries no
// migration params.
type migrationConfig struct {
	// enabled is the master switch. When false, the step runs its original
	// single-attempt path and the other fields are inert.
	enabled bool
	// maxConnectFailures bounds retries when the chosen worker never streams a
	// token (connect failure). 0 returns the failure to the client.
	maxConnectFailures int
	// maxRequestMigrations bounds mid-stream migrations after partial output has
	// streamed. 0 preserves existing behavior: the stream error reaches the
	// client. Meaningful only on the decode step.
	maxRequestMigrations int
	// maxMigratableTokens caps prompt + output tokens cached for a possible
	// migration. 0 means no cap.
	maxMigratableTokens int
}

// parseMigrationConfig reads the migration params from a step's config map. A
// negative bound is a configuration error. Absent keys leave the zero value,
// so a step with no migration params gets migration disabled.
func parseMigrationConfig(params map[string]any) (migrationConfig, error) {
	var cfg migrationConfig

	enabled, ok, err := paramBool(params, ParamEnableRequestMigration)
	if err != nil {
		return cfg, err
	}
	if ok {
		cfg.enabled = enabled
	}

	if cfg.maxConnectFailures, err = nonNegativeParamInt(params, ParamMaxConnectFailures); err != nil {
		return cfg, err
	}
	if cfg.maxRequestMigrations, err = nonNegativeParamInt(params, ParamMaxRequestMigrations); err != nil {
		return cfg, err
	}
	if cfg.maxMigratableTokens, err = nonNegativeParamInt(params, ParamMaxMigratableTokens); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// nonNegativeParamInt reads an integer param, defaulting to 0 when absent and
// rejecting a negative value (a bound cannot be negative).
func nonNegativeParamInt(params map[string]any, key string) (int, error) {
	v, ok, err := paramInt(params, key)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}
	if v < 0 {
		return 0, fmt.Errorf("%s: must not be negative, got %d", key, v)
	}
	return v, nil
}
