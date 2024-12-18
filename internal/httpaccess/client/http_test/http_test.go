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

package httpgw

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mkrainbow/rtio/pkg/rtioutil"
)

// Before run this test, please run the following program first.
// ./out/examples/deviceservice # option
// ./out/examples/hubconfiger	# option
// ./out/rtio  -disable.deviceverify -log.level debug -enable.jwt -jwt.ed25519  ./out/examples/certificates/ed25519.public
// ./out/devicehub/tcp_client
const (
	httpAddr = "0.0.0.0:17917"
)

type RTIOClient struct {
	client *http.Client
	url    string
	token  string
}

type JSONReq struct {
	Method string `json:"method"`
	ID     uint32 `json:"id"`
	URI    string `json:"uri"`
	Data   string `json:"data"`
}

type JSONResp struct {
	Code string `json:"code"`
	ID   uint32 `json:"id"`
	FID  uint32 `json:"fid"`
	Data string `json:"data"`
}

func NewRTIOClient(url, token string) *RTIOClient {
	httpTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: httpTransport, Timeout: 5 * time.Second}

	return &RTIOClient{
		client: client,
		url:    url,
		token:  token,
	}
}

func (c *RTIOClient) CoPost(jsonReq JSONReq) (*JSONResp, error) {
	req, err := json.Marshal(jsonReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.url, bytes.NewBuffer(req))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	resp, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	jsonResp := &JSONResp{}
	err = json.Unmarshal(resp, jsonResp)
	if err != nil {
		return nil, fmt.Errorf("%w,body=%s,reqlen=%d", err, resp, len(req))
	}
	return jsonResp, nil
}
func (c *RTIOClient) CoPostBytes(reqBody []byte) (*JSONResp, error) {

	httpReq, err := http.NewRequest("POST", c.url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	resp, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	jsonResp := &JSONResp{}
	err = json.Unmarshal(resp, jsonResp)
	if err != nil {
		return nil, fmt.Errorf("%w,body=%s,reqlen=%d", err, resp, len(reqBody))
	}
	return jsonResp, nil
}

func (c *RTIOClient) ObGet(t *testing.T, jsonReq JSONReq) (<-chan *JSONResp, error) {
	req, err := json.Marshal(jsonReq)
	if err != nil {
		return nil, err
	}
	t.Logf("reqlen=%d\n", len(req))

	httpReq, err := http.NewRequest("POST", c.url, bytes.NewBuffer(req))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(httpResp.Body)
	jsonRespChan := make(chan *JSONResp, 1)

	go func() {
		for {
			col, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					t.Log("stream EOF")
				} else {
					t.Error(err)
				}
				close(jsonRespChan)
				httpResp.Body.Close()
				return
			}

			jsonResp := &JSONResp{}
			err = json.Unmarshal([]byte(col), jsonResp)
			if err != nil {
				t.Errorf("%v,col=%s", err, col)
				return
			}
			jsonRespChan <- jsonResp
		}
	}()

	return jsonRespChan, nil
}

func TestObGet_DeviceOnLine(t *testing.T) {
	id, _ := rtioutil.GenUint32ID()
	base64Data := base64.StdEncoding.EncodeToString([]byte("hello"))
	t.Logf("base64Data=%v\n", base64Data)

	jsonReq := JSONReq{
		Method: "obget",
		ID:     id,
		URI:    "/test",
		Data:   base64Data,
	}
	t.Logf("jsonReq=%v\n", jsonReq)

	url := "http://" + httpAddr + "/cfa09baa-4913-4ad7-a936-2e26f9671b05"
	token := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE4ODAyNzg5MDgsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0yZTI2Zjk2NzFiMDUifQ.jclIibY2C6kU9FT7_VR5CcPUoKCrark7vYzdHiRac8JKIDKx3pk3Q9Knz4qhLDnzRB6JCMyQhwr__Nn-lvQtBQ"

	c := NewRTIOClient(url, token)
	jsonRespChan, err := c.ObGet(t, jsonReq)
	if err != nil {
		t.Errorf("err=%v\n", err)
	}

	for v := range jsonRespChan {
		t.Log(v)
		if v.Code != "CONTINUE" && v.Code != "TERMINATE" {
			t.Errorf("code=%s\n", v.Code)
		}
	}
}

func TestCoPost_DeviceOnLine(t *testing.T) {

	id, _ := rtioutil.GenUint32ID()
	base64Data := base64.StdEncoding.EncodeToString([]byte("hello"))
	t.Logf("base64Data=%v\n", base64Data)

	jsonReq := JSONReq{
		Method: "copost",
		ID:     id,
		URI:    "/test",
		Data:   base64Data,
	}
	t.Logf("jsonReq=%v\n", jsonReq)

	url := "http://" + httpAddr + "/cfa09baa-4913-4ad7-a936-2e26f9671b05"
	token := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE4ODAyNzg5MDgsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0yZTI2Zjk2NzFiMDUifQ.jclIibY2C6kU9FT7_VR5CcPUoKCrark7vYzdHiRac8JKIDKx3pk3Q9Knz4qhLDnzRB6JCMyQhwr__Nn-lvQtBQ"

	c := NewRTIOClient(url, token)
	jsonResp, err := c.CoPost(jsonReq)

	if err != nil {
		t.Errorf("err=%v\n", err)
	}
	t.Logf("jsonResp=%v\n", jsonResp)

	if jsonResp.Code != "OK" {
		t.Errorf("code=%v\n", jsonResp.Code)
	}

	if len(jsonReq.Data) > 0 {
		data, err := base64.StdEncoding.DecodeString(jsonReq.Data)
		if err != nil {
			t.Errorf("err=%v\n", err)
		}
		t.Logf("data=%v\n", string(data))

		if string(data) != "hello" {
			t.Error("data error")
		}

	} else {
		t.Error("data empty")
	}
}

func TestCoPost_DeviceOnLine_JWT_SignError(t *testing.T) {

	id, _ := rtioutil.GenUint32ID()
	base64Data := base64.StdEncoding.EncodeToString([]byte("hello"))
	t.Logf("base64Data=%v\n", base64Data)

	jsonReq := JSONReq{
		Method: "copost",
		ID:     id,
		URI:    "/test",
		Data:   base64Data,
	}
	t.Logf("jsonReq=%v\n", jsonReq)

	url := "http://" + httpAddr + "/cfa09baa-4913-4ad7-a936-2e26f9671b05"

	// modify a char
	token := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE4ODAyNzg5MDgsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0yZTI2Zjk2NzFiMDUifQ.jclIibY2C6kU9FT7_VR5CcPUoKCrark7vYzdHiRac8JKIDKx3pk3Q9Knz4qhLDnzRB6JCMyQhwr__Nn-lvQtqQ"

	c := NewRTIOClient(url, token)
	resp, err := c.CoPost(jsonReq)
	t.Log("resp=", resp)

	if err == nil {
		t.Error("err is nil")
	} else {
		if !strings.Contains(err.Error(), "invalid character") {
			t.Error("err is", err.Error())
		}
	}
}

func TestCoPost_DeviceOnLine_JWT_SignExpired(t *testing.T) {

	id, _ := rtioutil.GenUint32ID()
	base64Data := base64.StdEncoding.EncodeToString([]byte("hello"))
	t.Logf("base64Data=%v\n", base64Data)

	jsonReq := JSONReq{
		Method: "copost",
		ID:     id,
		URI:    "/test",
		Data:   base64Data,
	}
	t.Logf("jsonReq=%v\n", jsonReq)

	url := "http://" + httpAddr + "/cfa09baa-4913-4ad7-a936-2e26f9671b05"
	// expires = 2 seconds
	token := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjI2MDMyNTYsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0yZTI2Zjk2NzFiMDUifQ.Oz4tlq_dBmruLEbz2mjjYrFVlj-Uokp55mJiKJqanE-CUmrvPlmyO6gyrb7W6VF33VFoI8V4D7cY_-xU3XrVCA"

	c := NewRTIOClient(url, token)
	_, err := c.CoPost(jsonReq)

	t.Log(err)

	if err == nil {
		t.Error("err is nil")
	} else {
		if !strings.Contains(err.Error(), "invalid character") {
			t.Error("err is", err.Error())
		}
	}
}

func TestCoPost_DeviceOnLine_URI_LEN_128(t *testing.T) {

	id, _ := rtioutil.GenUint32ID()
	base64Data := base64.StdEncoding.EncodeToString([]byte("hello"))
	t.Logf("base64Data=%v\n", base64Data)

	jsonReq := JSONReq{
		Method: "copost",
		ID:     id,
		URI:    "/0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456",
		Data:   base64Data,
	}
	t.Logf("jsonReq=%v\n", jsonReq)

	url := "http://" + httpAddr + "/cfa09baa-4913-4ad7-a936-2e26f9671b05"
	token := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE4ODAyNzg5MDgsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0yZTI2Zjk2NzFiMDUifQ.jclIibY2C6kU9FT7_VR5CcPUoKCrark7vYzdHiRac8JKIDKx3pk3Q9Knz4qhLDnzRB6JCMyQhwr__Nn-lvQtBQ"

	c := NewRTIOClient(url, token)
	jsonResp, err := c.CoPost(jsonReq)

	if err != nil {
		t.Fatalf("err=%v\n", err)
	}
	t.Logf("jsonResp=%v\n", jsonResp)

	if jsonResp.Code != "OK" {
		t.Errorf("code=%v\n", jsonResp.Code)
	}
}
func TestCoPost_DeviceOnLine_URI_LEN_128More(t *testing.T) {

	id, _ := rtioutil.GenUint32ID()
	base64Data := base64.StdEncoding.EncodeToString([]byte("hello"))
	t.Logf("base64Data=%v\n", base64Data)

	jsonReq := JSONReq{
		Method: "copost",
		ID:     id,
		URI:    "/01234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567",
		Data:   base64Data,
	}
	t.Logf("jsonReq=%v\n", jsonReq)

	url := "http://" + httpAddr + "/cfa09baa-4913-4ad7-a936-2e26f9671b05"
	token := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE4ODAyNzg5MDgsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0yZTI2Zjk2NzFiMDUifQ.jclIibY2C6kU9FT7_VR5CcPUoKCrark7vYzdHiRac8JKIDKx3pk3Q9Knz4qhLDnzRB6JCMyQhwr__Nn-lvQtBQ"

	c := NewRTIOClient(url, token)
	jsonResp, err := c.CoPost(jsonReq)

	if err != nil {
		t.Fatalf("err=%v\n", err)
	}
	t.Logf("jsonResp=%v\n", jsonResp)

	if jsonResp.Code != "BAD_REQUEST" {
		t.Errorf("code=%v\n", jsonResp.Code)
	}
}
func TestCoPost_DeviceOnLine_URI_LEN_2(t *testing.T) {

	id, _ := rtioutil.GenUint32ID()
	base64Data := base64.StdEncoding.EncodeToString([]byte("hello"))
	t.Logf("base64Data=%v\n", base64Data)

	jsonReq := JSONReq{
		Method: "copost",
		ID:     id,
		URI:    "/0",
		Data:   base64Data,
	}
	t.Logf("jsonReq=%v\n", jsonReq)

	url := "http://" + httpAddr + "/cfa09baa-4913-4ad7-a936-2e26f9671b05"
	token := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE4ODAyNzg5MDgsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0yZTI2Zjk2NzFiMDUifQ.jclIibY2C6kU9FT7_VR5CcPUoKCrark7vYzdHiRac8JKIDKx3pk3Q9Knz4qhLDnzRB6JCMyQhwr__Nn-lvQtBQ"

	c := NewRTIOClient(url, token)
	jsonResp, err := c.CoPost(jsonReq)

	if err != nil {
		t.Fatalf("err=%v\n", err)
	}
	t.Logf("jsonResp=%v\n", jsonResp)

	if jsonResp.Code != "BAD_REQUEST" {
		t.Errorf("code=%v\n", jsonResp.Code)
	}
}
func TestCoPost_DeviceOnLine_Data_LEN_504(t *testing.T) {

	id, _ := rtioutil.GenUint32ID()

	// 504
	data := []byte("012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345670123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456701234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789")

	base64Data := base64.StdEncoding.EncodeToString(data)
	t.Logf("datalen=%v base64len=%v\n", len(data), len(base64Data))

	jsonReq := JSONReq{
		Method: "copost",
		ID:     id,
		URI:    "/test",
		Data:   base64Data,
	}
	t.Logf("jsonReq=%v\n", jsonReq)

	url := "http://" + httpAddr + "/cfa09baa-4913-4ad7-a936-2e26f9671b05"
	token := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE4ODAyNzg5MDgsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0yZTI2Zjk2NzFiMDUifQ.jclIibY2C6kU9FT7_VR5CcPUoKCrark7vYzdHiRac8JKIDKx3pk3Q9Knz4qhLDnzRB6JCMyQhwr__Nn-lvQtBQ"

	c := NewRTIOClient(url, token)
	jsonResp, err := c.CoPost(jsonReq)

	if err != nil {
		t.Fatalf("err=%v\n", err)
	}
	t.Logf("jsonResp=%v\n", jsonResp)

	if jsonResp.Code != "OK" {
		t.Errorf("code=%v\n", jsonResp.Code)
	}
}
func TestCoPost_DeviceOnLine_Data_LEN_505(t *testing.T) {

	id, _ := rtioutil.GenUint32ID()

	// 505
	data := []byte("9012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345670123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456701234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789")

	base64Data := base64.StdEncoding.EncodeToString(data)
	t.Logf("datalen=%v base64len=%v\n", len(data), len(base64Data))

	jsonReq := JSONReq{
		Method: "copost",
		ID:     id,
		URI:    "/test",
		Data:   base64Data,
	}
	t.Logf("jsonReq=%v\n", jsonReq)

	url := "http://" + httpAddr + "/cfa09baa-4913-4ad7-a936-2e26f9671b05"
	token := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE4ODAyNzg5MDgsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0yZTI2Zjk2NzFiMDUifQ.jclIibY2C6kU9FT7_VR5CcPUoKCrark7vYzdHiRac8JKIDKx3pk3Q9Knz4qhLDnzRB6JCMyQhwr__Nn-lvQtBQ"

	c := NewRTIOClient(url, token)
	jsonResp, err := c.CoPost(jsonReq)

	if err != nil {
		t.Fatalf("err=%v\n", err)
	}
	t.Logf("jsonResp=%v\n", jsonResp)

	if jsonResp.Code != "BAD_REQUEST" {
		t.Errorf("code=%v\n", jsonResp.Code)
	}
}

func TestCoPost_DeviceOnLine_LEN_REQ_864(t *testing.T) {

	id, _ := rtioutil.GenUint32ID()

	// 504
	data := []byte("01234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345670123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456701234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678")

	base64Data := base64.StdEncoding.EncodeToString(data)
	t.Logf("datalen=%v base64len=%v\n", len(data), len(base64Data))

	jsonReq := JSONReq{
		Method: "1234567890123456", // 17
		ID:     id,
		URI:    "/0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456",
		Data:   base64Data,
	}

	reqBody, _ := json.Marshal(jsonReq)
	t.Logf("reqBody=%v, len=%d\n", reqBody, len(reqBody)) // json.Marshal sometimes 864 bytes somtimes 863 bytes

	url := "http://" + httpAddr + "/cfa09baa-4913-4ad7-a936-2e26f9671b05"
	token := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE4ODAyNzg5MDgsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0yZTI2Zjk2NzFiMDUifQ.jclIibY2C6kU9FT7_VR5CcPUoKCrark7vYzdHiRac8JKIDKx3pk3Q9Knz4qhLDnzRB6JCMyQhwr__Nn-lvQtBQ"

	c := NewRTIOClient(url, token)
	jsonResp, err := c.CoPost(jsonReq)

	if err != nil {
		t.Fatalf("err=%v\n", err)
	}
	t.Logf("jsonResp=%v\n", jsonResp)

	if jsonResp.Code != "METHOD_NOT_ALLOWED" {
		t.Errorf("code=%v\n", jsonResp.Code)
	}
}

func TestCoPost_DeviceOnLine_LEN_REQ_865(t *testing.T) {

	reqBody := []byte("1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345")
	t.Logf("len=%d\n", len(reqBody))

	url := "http://" + httpAddr + "/cfa09baa-4913-4ad7-a936-2e26f9671b05"
	token := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE4ODAyNzg5MDgsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0yZTI2Zjk2NzFiMDUifQ.jclIibY2C6kU9FT7_VR5CcPUoKCrark7vYzdHiRac8JKIDKx3pk3Q9Knz4qhLDnzRB6JCMyQhwr__Nn-lvQtBQ"

	c := NewRTIOClient(url, token)
	resp, err := c.CoPostBytes(reqBody)
	t.Log("resp=", resp)

	if err == nil {
		t.Error("err is nil")
	} else {
		if !strings.Contains(err.Error(), "invalid character") {
			t.Error("err is", err.Error())
		}
	}
}

// ------------------------------------------------------------------------
//  DeviceOffLine test, using other deviceID(cfa09baa-4913-4ad7-a936-000000000000) and jwt.

func TestCoPost_DeviceOffLine(t *testing.T) {
	id, _ := rtioutil.GenUint32ID()
	base64Data := base64.StdEncoding.EncodeToString([]byte("hello"))
	t.Logf("base64Data=%v\n", base64Data)

	jsonReq := JSONReq{
		Method: "copost",
		ID:     id,
		URI:    "/test",
		Data:   base64Data,
	}
	t.Logf("jsonReq=%v\n", jsonReq)

	url := "http://" + httpAddr + "/cfa09baa-4913-4ad7-a936-000000000000"
	token := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE4OTAwMjQ0MjMsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0wMDAwMDAwMDAwMDAifQ.pxxJk12qz2DyQ1IZX7r67kJOVOYC43ZES-i1F2ZpdyN3jmb32XS6zR6bkFp-ApCBa3HxxJwW0eGRkbRBec62AA"

	c := NewRTIOClient(url, token)
	jsonResp, err := c.CoPost(jsonReq)

	if err != nil {
		t.Errorf("err=%v\n", err)
	}
	t.Logf("jsonResp=%v\n", jsonResp)

	if jsonResp.Code != "DEVICEID_OFFLINE" {
		t.Errorf("code=%v\n", jsonResp.Code)
	}
}

func TestObGet_DeviceOffLine(t *testing.T) {
	id, _ := rtioutil.GenUint32ID()
	base64Data := base64.StdEncoding.EncodeToString([]byte("hello"))
	t.Logf("base64Data=%v\n", base64Data)

	jsonReq := JSONReq{
		Method: "obget",
		ID:     id,
		URI:    "/test",
		Data:   base64Data,
	}
	t.Logf("jsonReq=%v\n", jsonReq)

	url := "http://" + httpAddr + "/cfa09baa-4913-4ad7-a936-000000000000"
	token := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE4OTAwMjQ0MjMsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0wMDAwMDAwMDAwMDAifQ.pxxJk12qz2DyQ1IZX7r67kJOVOYC43ZES-i1F2ZpdyN3jmb32XS6zR6bkFp-ApCBa3HxxJwW0eGRkbRBec62AA"

	c := NewRTIOClient(url, token)
	jsonRespChan, err := c.ObGet(t, jsonReq)
	if err != nil {
		t.Errorf("err=%v\n", err)
	}

	for v := range jsonRespChan {
		t.Log(v)
		if v.Code != "DEVICEID_OFFLINE" {
			t.Errorf("code=%s\n", v.Code)
		}
	}
}
