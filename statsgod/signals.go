/**
 * Copyright 2015 Acquia, Inc.
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

// Package statsgod - This library is responsible for parsing and defining what
// a "metric" is that we are going to be aggregating.
package statsgod

import (
	"os"
	"os/signal"
	"syscall"
)

// ListenForSignals will listen for quit/reload signals and respond accordingly.
func ListenForSignals(finishChannel chan int, config *ConfigValues, configFile *string, logger Logger) {
	// For signal handling we catch several signals. ABRT, INT, TERM and QUIT
	// are all used to clean up and stop the process. HUP is used to signal
	// a configuration reload without stopping the process.
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel,
		syscall.SIGABRT,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		for {
			s := <-signalChannel
			logger.Info.Printf("Processed signal %v", s)

			switch s {
			case syscall.SIGHUP:
				// Reload the configuration.
				logger.Info.Printf("Loading config changes from %s", *configFile)
				config.LoadFile(*configFile)
			default:
				// The other signals will kill the process.
				finishChannel <- 1
				break
			}
		}
	}()

}
