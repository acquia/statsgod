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

// Package statsgod - This library handles the statistics calculations.
package statsgod

import (
	"errors"
	"math"
	"sort"
)

// ValueSlice provides a storage for float64 values.
type ValueSlice []float64

// Get is a getter for internal values.
func (values ValueSlice) Get(i int) float64 { return values[i] }

// Len gets the length of the internal values.
func (values ValueSlice) Len() int { return len(values) }

// Less is used for the sorting interface.
func (values ValueSlice) Less(i, j int) bool { return values[i] < values[j] }

// Swap is used for the sorting interface.
func (values ValueSlice) Swap(i, j int) { values[i], values[j] = values[j], values[i] }

// UniqueCount provides the number of unique values sent during this period.
func (values ValueSlice) UniqueCount() int {
	sort.Sort(values)
	count := 0
	var p float64
	for i, v := range values {
		if i > 0 && p != v {
			count++
		} else if i == 0 {
			count++
		}
		p = v
	}

	return count
}

// Minmax provides the minimum and maximum values for a ValueSlice structure.
func (values ValueSlice) Minmax() (min float64, max float64, err error) {
	length := values.Len()
	min = values.Get(0)
	max = values.Get(0)

	for i := 0; i < length; i++ {
		xi := values.Get(i)

		if math.IsNaN(xi) {
			err = errors.New("NaN value")
		} else if xi < min {
			min = xi
		} else if xi > max {
			max = xi
		}
	}

	return
}

// Median finds the median value from a ValueSlice.
func (values ValueSlice) Median() (median float64) {
	length := values.Len()
	leftBoundary := (length - 1) / 2
	rightBoundary := length / 2

	if length == 0 {
		return 0.0
	}

	if leftBoundary == rightBoundary {
		median = values.Get(leftBoundary)
	} else {
		median = (values.Get(leftBoundary) + values.Get(rightBoundary)) / 2.0
	}

	return
}

// Mean finds the mean value from a ValueSlice.
func (values ValueSlice) Mean() (mean float64) {
	length := values.Len()
	for i := 0; i < length; i++ {
		mean += (values.Get(i) - mean) / float64(i+1)
	}
	return
}

// Quantile finds a specified quantile value from a ValueSlice.
func (values ValueSlice) Quantile(quantile float64) float64 {
	length := values.Len()
	index := quantile * float64(length-1)
	boundary := int(index)
	delta := index - float64(boundary)

	if length == 0 {
		return 0.0
	} else if boundary == length-1 {
		return values.Get(boundary)
	} else {
		return (1-delta)*values.Get(boundary) + delta*values.Get(boundary+1)
	}
}
