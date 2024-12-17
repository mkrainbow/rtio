/*
*
* Copyright 2023-2024 mkrainbow.com.
* 
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
* 
*      http://www.apache.org/licenses/LICENSE-2.0
* 
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
* 
*/

package verifier

import "testing"

func TestVerifyOK(t *testing.T) {
	c := NewClient("http://0.0.0.0:17217/deviceverifier")
	req := &VerifyReq{
		ID:           12345,
		Method:       "verify",
		DeviceID:     "cfa09baa-4913-4ad7-a936-2e26f9671b05",
		DeviceSecret: "mb6bgso4EChvyzA05thF9+wH",
	}

	resp, err := c.httpVerify(req)

	if err != nil {
		t.Fatalf("err=%v\n", err)
	}
	t.Log(resp)
	if resp.Code != "OK" {
		t.Errorf("code=%v\n", resp.Code)
	}
	if resp.ID != 12345 {
		t.Errorf("code=%v\n", resp.Code)
	}
}

func TestVerifyErrorMethod(t *testing.T) {
	c := NewClient("http://0.0.0.0:17217/deviceverifier")
	req := &VerifyReq{
		ID:           12345,
		Method:       "other",
		DeviceID:     "cfa09baa-4913-4ad7-a936-2e26f9671b05",
		DeviceSecret: "mb6bgso4EChvyzA05thF9+wH",
	}

	resp, err := c.httpVerify(req)
	if err != nil {
		t.Fatalf("err=%v\n", err)
	}
	t.Log(resp)
	if resp.Code != "METHOD_NOT_ALLOWED" {
		t.Errorf("code=%v\n", resp.Code)
	}
	if resp.ID != 12345 {
		t.Errorf("code=%v\n", resp.Code)
	}
}

func TestVerifyErrorSecret(t *testing.T) {
	c := NewClient("http://0.0.0.0:17217/deviceverifier")
	req := &VerifyReq{
		ID:           12345,
		Method:       "verify",
		DeviceID:     "cfa09baa-4913-4ad7-a936-2e26f9671b05",
		DeviceSecret: "mb6bgso4EChvyzA05thF9+wHa",
	}

	resp, err := c.httpVerify(req)
	if err != nil {
		t.Fatalf("err=%v\n", err)
	}
	t.Log(resp)
	if resp.Code != "VERIFICATION_FAILED" {
		t.Errorf("code=%v\n", resp.Code)
	}
	if resp.ID != 12345 {
		t.Errorf("code=%v\n", resp.Code)
	}
}
