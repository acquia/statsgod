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
	"math"
	"sort"
	"testing"
)

// TestValueSlice tests the ValueSlice struct.
func TestValueSliceStructure(t *testing.T) {
	values := ValueSlice{1, 5, 2, 4, 3}
	// Ensure we can get a correct length.
	assert.Equal(t, values.Len(), 5)
	// Ensure we can get a value at an index.
	assert.Equal(t, values.Get(3), 4)
	// Test the sort interface.
	sort.Sort(values)
	assert.Equal(t, values.Get(0), 1)
	assert.Equal(t, values.Get(1), 2)
	assert.Equal(t, values.Get(2), 3)
	assert.Equal(t, values.Get(3), 4)
	assert.Equal(t, values.Get(4), 5)
}

// TestMinmax tests the Minmax function.
func TestMinmax(t *testing.T) {
	values := ValueSlice{5, math.NaN(), 2, 3, 4, 1}
	min, max, err := values.Minmax()
	assert.Equal(t, min, 1)
	assert.Equal(t, max, 5)
	assert.NotNil(t, err)
}

// TestMedian tests the Median function.
func TestMedian(t *testing.T) {
	values := ValueSlice{123, 234, 345, 456, 567, 678, 789}
	median := values.Median()
	assert.Equal(t, median, 456)

	values = append(values, 890)
	median = values.Median()
	assert.Equal(t, median, 511.5)

	values = ValueSlice{}
	median = values.Median()
	assert.Equal(t, median, 0.0)
}

// TestMedian tests the Mean function.
func TestMean(t *testing.T) {
	values := ValueSlice{123, 234, 345, 456, 567, 678, 789, 890}
	mean := values.Mean()
	assert.Equal(t, mean, 510.25)
}

// TestQuantile tests the Quantile function.
func TestQuantile(t *testing.T) {
	values := ValueSlice{123, 234, 345, 456, 567, 678, 789, 890, 910, 1011}
	q100 := values.Quantile(1)
	assert.Equal(t, q100, 1011)
	q90 := values.Quantile(0.9)
	assert.Equal(t, q90, 920.1)
	q80 := values.Quantile(0.8)
	assert.Equal(t, q80, 894)
	q75 := values.Quantile(0.75)
	assert.Equal(t, q75, 864.75)
	q50 := values.Quantile(0.5)
	assert.Equal(t, q50, 622.5)
	q25 := values.Quantile(0.25)
	assert.Equal(t, q25, 372.75)

	values = ValueSlice{}
	q0 := values.Quantile(1)
	assert.Equal(t, q0, 0.0)
}
