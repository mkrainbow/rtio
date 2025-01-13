/*
*
* Copyright 2023-2025 mkrainbow.com.
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

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/mkrainbow/rtio/pkg/rtioutil"
)

type Client struct {
	client *http.Client
	url    string
}

type VerifyReq struct {
	ID           uint32 `json:"id"`
	Method       string `json:"method"`
	DeviceID     string `json:"deviceid"`
	DeviceSecret string `json:"devicesecret"`
}

type VerifyResp struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
}

func NewClient(url string) *Client {
	httpTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: httpTransport, Timeout: 5 * time.Second}

	return &Client{
		client: client,
		url:    url,
	}
}

func (c *Client) httpVerify(req *VerifyReq) (*VerifyResp, error) {
	buf, err := json.Marshal(*req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to Marshal req")
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.url, bytes.NewBuffer(buf))
	if err != nil {
		log.Error().Err(err).Msg("Failed to NewRequest")
		return nil, err
	}
	httpResp, err := c.client.Do(httpReq)

	if err != nil {
		log.Error().Err(err).Msg("Failed to post req")
		return nil, err
	}
	defer httpResp.Body.Close()
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read body")
		return nil, err
	}

	resp := &VerifyResp{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Failed to Marshal resp")
		return nil, err
	}

	return resp, nil
}

func (c *Client) Verify(deviceID, deviceSecret string) (bool, error) {
	id, err := rtioutil.GenUint32ID()
	if err != nil {
		log.Error().Err(err).Msg("GenUint32ID err")
		return false, err
	}
	req := &VerifyReq{
		ID:           id,
		Method:       "verify",
		DeviceID:     deviceID,
		DeviceSecret: deviceSecret,
	}

	resp, err := c.httpVerify(req)

	if err != nil {
		log.Error().Err(err).Msg("Error while call http verify")
		return false, err
	}

	if resp.Code == "OK" {
		return true, nil
	} else if resp.Code == "VERIFICATION_FAILED" {
		return false, nil
	} else if resp.Code == "NOT_FOUND" {
		log.Warn().Str("deviceid", deviceID).Msg("Not Found device")
		return false, nil
	}
	log.Error().Str("deviceid", deviceID).Str("code", resp.Code).Msg("Failed to verify device")
	return false, nil
}
