//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package segment

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestAdhocSegmentsWithType(t *testing.T) {

	tests := []struct {
		input         []byte
		output        [][]byte
		outputStrings []string
		outputTypes   []int
	}{
		{
			input: []byte("Now is the.\n End."),
			output: [][]byte{
				[]byte("Now"),
				[]byte(" "),
				[]byte("is"),
				[]byte(" "),
				[]byte("the"),
				[]byte("."),
				[]byte("\n"),
				[]byte(" "),
				[]byte("End"),
				[]byte("."),
			},
			outputStrings: []string{
				"Now",
				" ",
				"is",
				" ",
				"the",
				".",
				"\n",
				" ",
				"End",
				".",
			},
			outputTypes: []int{
				Letter,
				None,
				Letter,
				None,
				Letter,
				None,
				None,
				None,
				Letter,
				None,
			},
		},
		{
			input: []byte("3.5"),
			output: [][]byte{
				[]byte("3.5"),
			},
			outputStrings: []string{
				"3.5",
			},
			outputTypes: []int{
				Number,
			},
		},
		{
			input: []byte("cat3.5"),
			output: [][]byte{
				[]byte("cat3.5"),
			},
			outputStrings: []string{
				"cat3.5",
			},
			outputTypes: []int{
				Letter,
			},
		},
		{
			input: []byte("c"),
			output: [][]byte{
				[]byte("c"),
			},
			outputStrings: []string{
				"c",
			},
			outputTypes: []int{
				Letter,
			},
		},
		{
			input: []byte("こんにちは世界"),
			output: [][]byte{
				[]byte("こ"),
				[]byte("ん"),
				[]byte("に"),
				[]byte("ち"),
				[]byte("は"),
				[]byte("世"),
				[]byte("界"),
			},
			outputStrings: []string{
				"こ",
				"ん",
				"に",
				"ち",
				"は",
				"世",
				"界",
			},
			outputTypes: []int{
				Ideo,
				Ideo,
				Ideo,
				Ideo,
				Ideo,
				Ideo,
				Ideo,
			},
		},
		{
			input: []byte("你好世界"),
			output: [][]byte{
				[]byte("你"),
				[]byte("好"),
				[]byte("世"),
				[]byte("界"),
			},
			outputStrings: []string{
				"你",
				"好",
				"世",
				"界",
			},
			outputTypes: []int{
				Ideo,
				Ideo,
				Ideo,
				Ideo,
			},
		},
		{
			input: []byte("サッカ"),
			output: [][]byte{
				[]byte("サッカ"),
			},
			outputStrings: []string{
				"サッカ",
			},
			outputTypes: []int{
				Ideo,
			},
		},
	}

	for _, test := range tests {
		rv := make([][]byte, 0)
		rvstrings := make([]string, 0)
		rvtypes := make([]int, 0)
		segmenter := NewWordSegmenter(bytes.NewReader(test.input))
		// Set the split function for the scanning operation.
		for segmenter.Segment() {
			rv = append(rv, segmenter.Bytes())
			rvstrings = append(rvstrings, segmenter.Text())
			rvtypes = append(rvtypes, segmenter.Type())
		}
		if err := segmenter.Err(); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(rv, test.output) {
			t.Fatalf("expected:\n%#v\ngot:\n%#v\nfor: '%s'", test.output, rv, test.input)
		}
		if !reflect.DeepEqual(rvstrings, test.outputStrings) {
			t.Fatalf("expected:\n%#v\ngot:\n%#v\nfor: '%s'", test.outputStrings, rvstrings, test.input)
		}
		if !reflect.DeepEqual(rvtypes, test.outputTypes) {
			t.Fatalf("expeced:\n%#v\ngot:\n%#v\nfor: '%s'", test.outputTypes, rvtypes, test.input)
		}
	}

}

func TestUnicodeSegments(t *testing.T) {

	for _, test := range unicodeWordTests {
		rv := make([][]byte, 0)
		scanner := bufio.NewScanner(bytes.NewReader(test.input))
		// Set the split function for the scanning operation.
		scanner.Split(SplitWords)
		for scanner.Scan() {
			rv = append(rv, scanner.Bytes())
		}
		if err := scanner.Err(); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(rv, test.output) {
			t.Fatalf("expected:\n%#v\ngot:\n%#v\nfor: '%s'", test.output, rv, test.input)
		}
	}
}

// Tests borrowed from Scanner to test Segmenter

// slowReader is a reader that returns only a few bytes at a time, to test the incremental
// reads in Scanner.Scan.
type slowReader struct {
	max int
	buf io.Reader
}

func (sr *slowReader) Read(p []byte) (n int, err error) {
	if len(p) > sr.max {
		p = p[0:sr.max]
	}
	return sr.buf.Read(p)
}

// genLine writes to buf a predictable but non-trivial line of text of length
// n, including the terminal newline and an occasional carriage return.
// If addNewline is false, the \r and \n are not emitted.
func genLine(buf *bytes.Buffer, lineNum, n int, addNewline bool) {
	buf.Reset()
	doCR := lineNum%5 == 0
	if doCR {
		n--
	}
	for i := 0; i < n-1; i++ { // Stop early for \n.
		c := 'a' + byte(lineNum+i)
		if c == '\n' || c == '\r' { // Don't confuse us.
			c = 'N'
		}
		buf.WriteByte(c)
	}
	if addNewline {
		if doCR {
			buf.WriteByte('\r')
		}
		buf.WriteByte('\n')
	}
	return
}

func wrapSplitFuncAsSegmentFuncForTesting(splitFunc bufio.SplitFunc) SegmentFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, typ int, err error) {
		typ = 0
		advance, token, err = splitFunc(data, atEOF)
		return
	}
}

// Test that the line segmenter errors out on a long line.
func TestSegmentTooLong(t *testing.T) {
	const smallMaxTokenSize = 256 // Much smaller for more efficient testing.
	// Build a buffer of lots of line lengths up to but not exceeding smallMaxTokenSize.
	tmp := new(bytes.Buffer)
	buf := new(bytes.Buffer)
	lineNum := 0
	j := 0
	for i := 0; i < 2*smallMaxTokenSize; i++ {
		genLine(tmp, lineNum, j, true)
		j++
		buf.Write(tmp.Bytes())
		lineNum++
	}
	s := NewSegmenter(&slowReader{3, buf})
	// change to line segmenter for testing
	s.SetSegmenter(wrapSplitFuncAsSegmentFuncForTesting(bufio.ScanLines))
	s.MaxTokenSize(smallMaxTokenSize)
	j = 0
	for lineNum := 0; s.Segment(); lineNum++ {
		genLine(tmp, lineNum, j, false)
		if j < smallMaxTokenSize {
			j++
		} else {
			j--
		}
		line := tmp.Bytes()
		if !bytes.Equal(s.Bytes(), line) {
			t.Errorf("%d: bad line: %d %d\n%.100q\n%.100q\n", lineNum, len(s.Bytes()), len(line), s.Bytes(), line)
		}
	}
	err := s.Err()
	if err != ErrTooLong {
		t.Fatalf("expected ErrTooLong; got %s", err)
	}
}

var testError = errors.New("testError")

// Test the correct error is returned when the split function errors out.
func TestSegmentError(t *testing.T) {
	// Create a split function that delivers a little data, then a predictable error.
	numSplits := 0
	const okCount = 7
	errorSplit := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF {
			panic("didn't get enough data")
		}
		if numSplits >= okCount {
			return 0, nil, testError
		}
		numSplits++
		return 1, data[0:1], nil
	}
	// Read the data.
	const text = "abcdefghijklmnopqrstuvwxyz"
	buf := strings.NewReader(text)
	s := NewSegmenter(&slowReader{1, buf})
	// change to line segmenter for testing
	s.SetSegmenter(wrapSplitFuncAsSegmentFuncForTesting(errorSplit))
	var i int
	for i = 0; s.Segment(); i++ {
		if len(s.Bytes()) != 1 || text[i] != s.Bytes()[0] {
			t.Errorf("#%d: expected %q got %q", i, text[i], s.Bytes()[0])
		}
	}
	// Check correct termination location and error.
	if i != okCount {
		t.Errorf("unexpected termination; expected %d tokens got %d", okCount, i)
	}
	err := s.Err()
	if err != testError {
		t.Fatalf("expected %q got %v", testError, err)
	}
}

// Test that Scan finishes if we have endless empty reads.
type endlessZeros struct{}

func (endlessZeros) Read(p []byte) (int, error) {
	return 0, nil
}

func TestBadReader(t *testing.T) {
	scanner := NewSegmenter(endlessZeros{})
	for scanner.Segment() {
		t.Fatal("read should fail")
	}
	err := scanner.Err()
	if err != io.ErrNoProgress {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSegmentAdvanceNegativeError(t *testing.T) {
	errorSplit := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF {
			panic("didn't get enough data")
		}
		return -1, data[0:1], nil
	}
	// Read the data.
	const text = "abcdefghijklmnopqrstuvwxyz"
	buf := strings.NewReader(text)
	s := NewSegmenter(&slowReader{1, buf})
	// change to line segmenter for testing
	s.SetSegmenter(wrapSplitFuncAsSegmentFuncForTesting(errorSplit))
	s.Segment()
	err := s.Err()
	if err != ErrNegativeAdvance {
		t.Fatalf("expected %q got %v", testError, err)
	}
}

func TestSegmentAdvanceTooFarError(t *testing.T) {
	errorSplit := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF {
			panic("didn't get enough data")
		}
		return len(data) + 10, data[0:1], nil
	}
	// Read the data.
	const text = "abcdefghijklmnopqrstuvwxyz"
	buf := strings.NewReader(text)
	s := NewSegmenter(&slowReader{1, buf})
	// change to line segmenter for testing
	s.SetSegmenter(wrapSplitFuncAsSegmentFuncForTesting(errorSplit))
	s.Segment()
	err := s.Err()
	if err != ErrAdvanceTooFar {
		t.Fatalf("expected %q got %v", testError, err)
	}
}
