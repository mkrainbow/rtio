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

package configer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"hash/crc32"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/mkrainbow/rtio/pkg/config"
	"github.com/mkrainbow/rtio/pkg/rtioutil"
)

var (
	ErrRequestIDNotMatch = errors.New("Failed to get device verify client")

	currentConfigDigest uint32
)

type Config struct {
	DeviceServiceMap map[string]string `json:"deviceservicemap"`
}

type Client struct {
	client *http.Client
	url    string
}

type ConfigReq struct {
	ID           uint32 `json:"id"`
	Method       string `json:"method"`
	DeviceID     string `json:"deviceid"`
	DeviceSecret string `json:"devicesecret"`
}

type ConfigResp struct {
	ID     uint32 `json:"id"`
	Code   string `json:"code"`
	Config string `json:"config"`
	Digest uint32 `json:"digest"`
}

func newHttpClient(url string) *Client {
	httpTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: httpTransport, Timeout: 5 * time.Second}

	return &Client{
		client: client,
		url:    url,
	}
}

func (c *Client) postRequest(req *ConfigReq) (*ConfigResp, error) {
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

	resp := &ConfigResp{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("Failed to Marshal resp")
		return nil, err
	}

	return resp, nil
}

func (c *Client) GetConfig() (uint32, string, error) {
	id, err := rtioutil.GenUint32ID()
	if err != nil {
		log.Error().Err(err).Msg("GenUint32ID err")
		return 0, "", err
	}
	req := &ConfigReq{
		ID:     id,
		Method: "getconfig",
	}
	resp, err := c.postRequest(req)
	if err != nil {
		log.Error().Err(err).Msg("Error while call http verify")
		return 0, "", err
	}
	if resp.ID != req.ID {
		return 0, "", ErrRequestIDNotMatch
	}
	if resp.Code != "OK" {
		log.Error().Str("code", resp.Code).Msg("Failed to call hubconfiger")
		return 0, "", nil
	}
	digest := crc32.ChecksumIEEE([]byte(resp.Config))
	if resp.Digest != digest {
		log.Error().Uint32("resp.digest", resp.Digest).Uint32("digest", digest).Msg("Digest not Match")
		log.Debug().Uint32("resp.digest", resp.Digest).Str("resp.config", resp.Config).Msg("Digest not Match")
		return 0, "", nil
	}
	// log.Debug().Uint32("resp.digest", resp.Digest).Msg("Config is complete")
	return digest, resp.Config, nil
}

func tryUpdateConfig(c *Client) {

	digest, configText, err := c.GetConfig()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get config")
	}

	if digest != currentConfigDigest {
		c := &Config{}
		err = json.Unmarshal([]byte(configText), c)
		if err != nil {
			log.Error().Err(err).Msg("Failed to Unmarshal")
			return
		}

		digest := crc32.ChecksumIEEE([]byte(configText))
		log.Info().Uint32("old", currentConfigDigest).Uint32("new", digest).Msg("Update config...")
		currentConfigDigest = digest

		for k, v := range c.DeviceServiceMap {
			d := crc32.ChecksumIEEE([]byte(k))
			config.StringKV.Set("deviceservice."+strconv.FormatUint(uint64(d), 16), v)
		}

		// show configs
		for _, v := range config.StringKV.List() {
			log.Debug().Msgf("Config: %s", v)
		}
	}
}

func HubConfigerInit(ctx context.Context, wait *sync.WaitGroup) {

	currentConfigDigest = 0
	url, ok := config.StringKV.Get("backend.hubconfiger")
	if !ok {
		log.Error().Msg("hub configer URL empty, hubconfiger route exit")
		return
	}
	httpclient := newHttpClient(url)

	t := time.NewTicker(time.Second * 5)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("ctx done")
			return
		case <-t.C:
			tryUpdateConfig(httpclient)
		}
	}
}
