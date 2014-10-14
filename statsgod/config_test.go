/**
 * Copyright 2014 Acquia, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package statsgod

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// Tests all of the known configuration values and types.
func TestLoadConfig(t *testing.T) {
	// Ensure we can get default values by loading without a file. Then
	// ensure that the default config file matches those values.
	config, _ := LoadConfig("")
	yaml, _ := LoadConfig("../config.yml")
	assert.Equal(t, config.Service.Name, yaml.Service.Name)
	assert.Equal(t, config.Service.Debug, yaml.Service.Debug)
	assert.Equal(t, config.Connection.Tcp.Host, yaml.Connection.Tcp.Host)
	assert.Equal(t, config.Connection.Tcp.Port, yaml.Connection.Tcp.Port)
	assert.Equal(t, config.Connection.Udp.Host, yaml.Connection.Udp.Host)
	assert.Equal(t, config.Connection.Udp.Port, yaml.Connection.Udp.Port)
	assert.Equal(t, config.Connection.Unix.File, yaml.Connection.Unix.File)
	assert.Equal(t, config.Relay.Type, yaml.Relay.Type)
	assert.Equal(t, config.Relay.Concurrency, yaml.Relay.Concurrency)
	assert.Equal(t, config.Relay.Timeout, yaml.Relay.Timeout)
	assert.Equal(t, config.Relay.Flush, yaml.Relay.Flush)
	assert.Equal(t, config.Carbon.Host, yaml.Carbon.Host)
	assert.Equal(t, config.Carbon.Port, yaml.Carbon.Port)
	assert.Equal(t, config.Stats.Percentile, yaml.Stats.Percentile)

	// Test that a non-existent file throws an error
	_, noFileErr := LoadConfig("noFile")
	assert.NotNil(t, noFileErr)
}
