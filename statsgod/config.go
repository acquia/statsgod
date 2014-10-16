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

// Package statsgod - This library handles the file-based runtime configuration.
package statsgod

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

// ConfigValues describes the data type that configuration is loaded into. The
// values from the config file map directly to these values. e.g.
//   service:
//       name: statsgod
//       debug: true
// All values specified in the ConfigValues struct should also have a default
// value set in LoadConfig() to ensure a safe runtime environment.
type ConfigValues struct {
	Service struct {
		Name  string
		Debug bool
	}
	Connection struct {
		Tcp struct {
			Host string
			Port int
		}
		Udp struct {
			Host string
			Port int
		}
		Unix struct {
			File string
		}
	}
	Relay struct {
		Type        string
		Concurrency int
		Timeout     time.Duration
	}
	Flush struct {
		Interval        time.Duration
		PersistDuration time.Duration
	}
	Carbon struct {
		Host string
		Port int
	}
	Stats struct {
		Percentile int
	}
}

// LoadConfig will read configuration from a specified file.
func LoadConfig(filePath string) (config ConfigValues, err error) {
	// Establish all of the default values.
	config.Service.Name = "statsgod"
	config.Service.Debug = false
	config.Connection.Tcp.Host = "127.0.0.1"
	config.Connection.Tcp.Port = 8125
	config.Connection.Udp.Host = "127.0.0.1"
	config.Connection.Udp.Port = 8126
	config.Connection.Unix.File = "/var/run/statsgod/statsgod.sock"
	config.Relay.Type = "carbon"
	config.Relay.Concurrency = 1
	config.Relay.Timeout = 30 * time.Second
	config.Flush.Interval = 10 * time.Second
	config.Flush.PersistDuration = 10 * time.Second
	config.Carbon.Host = "127.0.0.1"
	config.Carbon.Port = 2003
	config.Stats.Percentile = 80

	// Attempt to read in the file.
	if filePath != "" {
		contents, readError := ioutil.ReadFile(filePath)
		if readError != nil {
			err = readError
		} else {
			err = yaml.Unmarshal([]byte(contents), &config)
		}
	}

	return
}
