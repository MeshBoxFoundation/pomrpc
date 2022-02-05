// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"testing"
	"time"
)

func TestPosts(t *testing.T) {

	gethCmd := exec.Command("geth", "--dev", "--http", "--http.port", "8545")
	if err := gethCmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer gethCmd.Process.Kill()
	r, err := url.Parse("http://localhost:8545")
	if err != nil {
		t.Fatal(err)
	}
	remote = r

	type testCase struct {
		postData     jsonrpcMessage
		expectStatus int
	}
	cases := []testCase{
		{
			postData: jsonrpcMessage{
				ID:      []byte(fmt.Sprintf("%d", rand.Int())),
				Version: "2.0",
				Method:  "eth_blockNumber",
				Params:  []byte(`[]`),
			},
			expectStatus: http.StatusOK,
		},
		{
			postData: jsonrpcMessage{
				ID:      []byte(fmt.Sprintf("%d", rand.Int())),
				Version: "2.0",
				Method:  "eth_getBalance",
				Params:  []byte(`["0xDf7D7e053933b5cC24372f878c90E62dADAD5d42", "latest"]`),
			},
			expectStatus: http.StatusOK,
		},
		{
			postData: jsonrpcMessage{
				ID:      []byte(fmt.Sprintf("%d", rand.Int())),
				Version: "2.0",
				Method:  "eth_blockNoop",
				Params:  []byte(`[]`),
			},
			expectStatus: http.StatusOK,
		},
	}

	runTest := func(i int, c testCase) {
		b, err := json.Marshal(c.postData)
		if err != nil {
			t.Fatal(i, err)
		}

		req, err := http.NewRequest("POST", "/", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json; charset=UTF-8")
		if err != nil {
			t.Fatal(i, err)
		}

		reqDump, err := httputil.DumpRequest(req, true)
		if err != nil {
			t.Fatal(i, err)
		}
		t.Logf("-> %v", string(reqDump))

		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(handler)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != c.expectStatus {
			dump, _ := httputil.DumpResponse(rr.Result(), true)
			t.Logf("<- %v", string(dump))
			t.Errorf(
				"case: %d, unexpected status: got (%v) want (%v)",
				i,
				status,
				http.StatusOK,
			)
		} else {
			msg, err := responseToJSONRPC(rr.Result())
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(c.postData.ID, msg.ID) {
				t.Errorf("mismatched ids: request=%v response=%v", string(c.postData.ID), string(msg.ID))
			}

			dump, err := httputil.DumpResponse(rr.Result(), true)
			if err != nil {
				t.Errorf("dump error: %v", err)
			}
			t.Logf("<- %s", string(dump))

		}
	}

	for i, c := range cases {
		runTest(i, c)
		runTest(i, c)
		time.Sleep(defaultCacheExpiration)
		runTest(i, c)
		runTest(i, c)
	}
}

// TestHandlingMethodType expects that the server handles only POST methods.
func TestHandlingMethodType(t *testing.T) {

	req, err := http.NewRequest("GET", "/", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":5577006791947779410,"method":"eth_blockNumber","params":[]}`)))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(handler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		dump, _ := httputil.DumpResponse(rr.Result(), true)
		t.Logf("<- %v", string(dump))
		t.Errorf(
			"unexpected status: got (%v) want (%v)",
			status,
			http.StatusBadRequest,
		)
	}
}
