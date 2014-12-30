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

// Package statsgod - this library manages authentication methods. All auth
// currently happens at the metric level. This means that all authentication
// parameters must be included in the metric string itself before it is parsed.
package statsgod

import (
	"errors"
	"strings"
)

const (
	// AuthTypeNone defines a runtime with no authentication.
	AuthTypeNone = "none"
	// AuthTypeConfigToken is a token-based auth which is retrieved from
	// the main configuration file.
	AuthTypeConfigToken = "token"
)

// CreateAuth is a factory to create an Auth object.
func CreateAuth(config ConfigValues) Auth {
	switch config.Service.Auth {
	case AuthTypeConfigToken:
		tokenAuth := new(AuthConfigToken)
		tokenAuth.Tokens = config.Service.Tokens
		return tokenAuth

	case AuthTypeNone:
		fallthrough
	default:
		return new(AuthNone)
	}
}

// Auth is an interface describing statsgod authentication objects.
type Auth interface {
	// Authenticate takes a metric string and authenticates it. The metric
	// is passed by reference in case the Auth object needs to manipulate
	// it, such as removing a token.
	Authenticate(metric *string) (bool, error)
}

// AuthNone does nothing and allows all traffic.
type AuthNone struct {
}

// AuthConfigToken checks the configuration file for a valid auth token.
type AuthConfigToken struct {
	// Tokens contains a list of valid auth tokens with a token as the key
	// which maps to a boolean defining the validity of the token.
	Tokens map[string]bool
}

// Authenticate conforms to Auth.Authenticate()
func (a AuthNone) Authenticate(metric *string) (bool, error) {
	return true, nil
}

// Authenticate conforms to Auth.Authenticate()
func (a AuthConfigToken) Authenticate(metric *string) (bool, error) {
	authSplit := strings.SplitN(*metric, ".", 2)
	if len(authSplit) == 2 {
		token, exists := a.Tokens[authSplit[0]]
		if !exists || !token {
			authError := errors.New("Invalid authentication token")
			return false, authError
		}
	} else {
		authError := errors.New("Missing authentication token")
		return false, authError
	}
	*metric = authSplit[1]
	return true, nil
}
