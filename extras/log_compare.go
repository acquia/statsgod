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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

var logInputFile = flag.String("in", "/tmp/statsgod.input", "The stdout from loadtest.go")
var logOutputFile = flag.String("out", "/tmp/statsgod.output", "The stdout from statsgod.go")
var token = flag.String("token", "", "An auth token to prepend to the metric string.")

func main() {
	flag.Usage = func() {
		usage := `Usage of %s:

1. Start statsgod.go with config.debug.receipt set to true and redirect to a file:
go run statsgod.go 2>&1 /tmp/statsgod.output

2. Start loadtest.go with the -logSent flag and redirect to a file:
go run extras/loadtest.go [options] -logSent=true 2>&1 /tmp/statsgod.input

3. After collecting input and output, compare using this utility:
go run extras/log_compare.go -in=/tmp/statsgod.input -out=/tmp/statsgod.output

`
		fmt.Fprintf(os.Stderr, usage, os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	inputBytes, err := ioutil.ReadFile(*logInputFile)
	inputString := string(inputBytes)
	if err != nil {
		panic("Could not open input file.")
	}
	inputLines := strings.Split(inputString, "\n")
	inputStrings := make(map[string]int)

	outputBytes, err := ioutil.ReadFile(*logOutputFile)
	outputString := string(outputBytes)
	if err != nil {
		panic("Could not open output file.")
	}
	outputLines := strings.Split(outputString, "\n")
	outputStrings := make(map[string]int)
	var outputLine string

	inputLen := len(inputLines)
	outputLen := len(outputLines)
	totalLines := inputLen

	fmt.Printf("Lines in input file (%s): %d\n", *logInputFile, inputLen)
	fmt.Printf("Lines in output file (%s): %d\n", *logOutputFile, outputLen)

	var metricType string
	var metricKey string

	if inputLen < outputLen {
		totalLines = outputLen
	}

	outputRegex := regexp.MustCompile("^.*Metric: \\{([^\\s]+)\\s([^\\s]+)\\s.*[\\s|\\[]([0-9\\.]+)\\].*$")

	inputStringsLen := 0
	outputStringsLen := 0

	// If the user specified an auth token, construct it here.
	metricToken := ""
	if *token != "" {
		metricToken = *token + "\\."
	}

	for i := 0; i < totalLines; i++ {
		metricType = ""
		metricKey = ""

		if i < inputLen {
			inputR, _ := regexp.Match("^"+metricToken+"[^\\s]+\\:[0-9\\.]+\\|(c|g|s|ms).*$", []byte(inputLines[i]))
			if inputR {
				// Strip off the token if it exists as the output won't specify it.
				if metricToken != "" {
					metricKey = strings.Replace(inputLines[i], *token+".", "", 1)
				} else {
					metricKey = inputLines[i]
				}

				inputStringsLen++
				if inputStrings[metricKey] == 0 {
					inputStrings[metricKey] = 1
				} else {
					inputStrings[metricKey]++
				}
			}
		}

		if i < outputLen {
			outputR := outputRegex.FindAllStringSubmatch(outputLines[i], -1)
			if len(outputR) > 0 && len(outputR[0]) == 4 {
				switch outputR[0][2] {
				case "0":
					metricType = "c"
				case "1":
					metricType = "g"
				case "2":
					metricType = "s"
				case "3":
					metricType = "ms"
				}
				outputLine = fmt.Sprintf("%s:%s|%s", outputR[0][1], outputR[0][3], metricType)
				outputStringsLen++
				if outputStrings[outputLine] == 0 {
					outputStrings[outputLine] = 1
				} else {
					outputStrings[outputLine]++
				}
			}
		}

	}

	fmt.Printf("Input metrics: %d\n", inputStringsLen)
	fmt.Printf("Output metrics: %d\n", outputStringsLen)

	totalLines = inputStringsLen
	if inputStringsLen < outputStringsLen {
		totalLines = outputStringsLen
	}

	matchCount := 0
	errorCount := 0

	for inputMetric, inputCount := range inputStrings {
		if inputCount == outputStrings[inputMetric] {
			matchCount++
			delete(inputStrings, inputMetric)
			delete(outputStrings, inputMetric)
		} else {
			errorCount++
		}
	}

	fmt.Printf("Consolidated metric matches: %d\n", matchCount)
	fmt.Printf("Consolidated metric errors: %d\n", errorCount)
	fmt.Printf("Input not matched: %v\n", inputStrings)
	fmt.Printf("Output not matched: %v\n", outputStrings)
}
