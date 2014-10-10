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
	"reflect"
	"testing"
)

// TestCreateRelay tests CreateRelay().
func TestCreateRelay(t *testing.T) {

	// Tests that we can get a mock relay.
	mockRelay := CreateRelay("mock")
	assert.NotNil(t, mockRelay)
	assert.Equal(t, reflect.TypeOf(mockRelay).String(), "*statsgod.MockRelay")

	// Tests that we can get a carbon relay.
	carbonRelay := CreateRelay("carbon")
	assert.NotNil(t, carbonRelay)
	assert.Equal(t, reflect.TypeOf(carbonRelay).String(), "*statsgod.CarbonRelay")

	// Tests that we can get a mock relay as the default value
	fooRelay := CreateRelay("foo")
	assert.NotNil(t, fooRelay)
	assert.Equal(t, reflect.TypeOf(fooRelay).String(), "*statsgod.MockRelay")
}

// TestCarbonRelayStructure tests the CarbonRelay struct.
func TestCarbonRelayStructure(t *testing.T) {
	backendRelay := CreateRelay("carbon").(*CarbonRelay)
	assert.NotNil(t, backendRelay.FlushInterval)
	assert.NotNil(t, backendRelay.Percentile)
	// At this point the connection pool has not been established.
	assert.Nil(t, backendRelay.ConnectionPool)
}
