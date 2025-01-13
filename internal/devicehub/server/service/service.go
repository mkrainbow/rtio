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

package service

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type Client struct {
	client *http.Client
}

type RTIOReq struct {
	ID       uint32 `json:"id"`
	Method   string `json:"method"`
	DeviceID string `json:"deviceid"`
	Data     string `json:"data"`
}

type RTIOResp struct {
	ID   uint32 `json:"id"`
	Code string `json:"code"`
	Data string `json:"data"`
}

var (
	ErrBadRequest    = errors.New("Bad request")
	ErrServiceError  = errors.New("Service error")
	ErrInternelError = errors.New("Internel error")
)

func NewClient() *Client {
	httpTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: httpTransport, Timeout: 5 * time.Second}

	return &Client{
		client: client,
	}
}

func (c *Client) postRequest(url string, req *RTIOReq) (*RTIOResp, error) {
	buf, err := json.Marshal(*req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to Marshal req")
		return nil, ErrBadRequest
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(buf))
	if err != nil {
		log.Error().Err(err).Msg("Failed to NewRequest")
		return nil, ErrBadRequest
	}
	httpResp, err := c.client.Do(httpReq)

	if err != nil {
		log.Error().Err(err).Msg("Failed to post req")
		return nil, ErrServiceError
	}
	defer httpResp.Body.Close()
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read body")
		return nil, ErrServiceError
	}

	resp := &RTIOResp{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Failed to Marshal resp")
		return nil, ErrServiceError
	}

	return resp, nil
}

func (c *Client) Post(id uint32, url, deviceID string, reqData []byte) ([]byte, error) {
	req := &RTIOReq{
		ID:       id,
		Method:   "copost",
		DeviceID: deviceID,
		Data:     base64.StdEncoding.EncodeToString(reqData),
	}

	resp, err := c.postRequest(url, req)

	if err != nil {
		return nil, err
	}

	if resp.Code != "OK" {
		if resp.Code == "BAD_REQUEST" {
			return nil, ErrBadRequest
		}
		return nil, ErrInternelError // resp.Code == "INTERNAL_SERVER_ERROR or resp.Code == "METHOD_NOT_ALLOWED"
	}

	respData, err := base64.StdEncoding.DecodeString(resp.Data)
	if err != nil {
		log.Error().Err(err).Msg("Failed to decode resp")
		return nil, ErrServiceError
	}
	return respData, nil
}
