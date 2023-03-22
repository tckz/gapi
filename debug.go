// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Based on https://github.com/google/google-api-go-client/blob/master/examples/debug.go

package gapi

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
)

type LogTransport struct {
	Rt http.RoundTripper
}

func (t *LogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var buf bytes.Buffer

	os.Stdout.Write([]byte("\n[request]\n"))
	if req.Body != nil {
		req.Body = io.NopCloser(&readButCopy{req.Body, &buf})
	}
	req.Write(os.Stdout)
	if req.Body != nil {
		req.Body = io.NopCloser(&buf)
	}
	os.Stdout.Write([]byte("\n[/request]\n"))

	res, err := t.Rt.RoundTrip(req)

	fmt.Printf("[response]\n")
	if err != nil {
		fmt.Printf("ERROR: %v", err)
	} else {
		body := res.Body
		res.Body = nil
		res.Write(os.Stdout)
		if body != nil {
			res.Body = io.NopCloser(&echoAsRead{body})
		}
	}

	return res, err
}

type echoAsRead struct {
	src io.Reader
}

func (r *echoAsRead) Read(p []byte) (int, error) {
	n, err := r.src.Read(p)
	if n > 0 {
		os.Stdout.Write(p[:n])
	}
	if err == io.EOF {
		fmt.Printf("\n[/response]\n")
	}
	return n, err
}

type readButCopy struct {
	src io.Reader
	dst io.Writer
}

func (r *readButCopy) Read(p []byte) (int, error) {
	n, err := r.src.Read(p)
	if n > 0 {
		r.dst.Write(p[:n])
	}
	return n, err
}
