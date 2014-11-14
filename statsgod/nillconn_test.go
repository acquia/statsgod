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

package statsgod_test

import (
	. "github.com/acquia/statsgod/statsgod"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("NilConn", func() {

	Describe("Testing the basic structure", func() {
		Context("when creating a NilConn", func() {
			It("should be a complete structure", func() {
				// Test the NilConn implementation.
				nilConn := NilConn{}
				nilBuf := make([]byte, 0)
				nilTime := time.Now()
				Expect(func() { nilConn.Read(nilBuf) }).Should(Panic())
				Expect(func() { nilConn.Write(nilBuf) }).Should(Panic())
				Expect(func() { nilConn.Close() }).Should(Panic())
				Expect(func() { nilConn.LocalAddr() }).Should(Panic())
				Expect(func() { nilConn.RemoteAddr() }).Should(Panic())
				Expect(func() { nilConn.SetDeadline(nilTime) }).Should(Panic())
				Expect(func() { nilConn.SetReadDeadline(nilTime) }).Should(Panic())
				Expect(func() { nilConn.SetWriteDeadline(nilTime) }).Should(Panic())
			})
		})
	})
})
