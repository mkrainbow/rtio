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

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"hash/crc32"
	"io"
	"net/http"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	ErrHTTPBodyReadFailed           = errors.New("Read body failed")
	ErrHTTPBodyJsonUnmarshallFailed = errors.New("Read body json unmarshall failed")
	ErrHTTPJSONInvalidMethod        = errors.New("json req method error")
)

const (
	RTIOCodeInternalServerError = "INTERNAL_SERVER_ERROR"
	RTIOCodeOk                  = "OK"
	RTIOCodeNotFound            = "NOT_FOUND"
	RTIOCodeBadRequest          = "BAD_REQUEST"
	RTIOCodeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
)

type RTIOReq struct {
	ID     uint32 `json:"id"`
	Method string `json:"method"`
}

type RTIOResp struct {
	ID     uint32 `json:"id"`
	Code   string `json:"code"`
	Config string `json:"config"`
	Digest uint32 `json:"digest"`
}

type Config struct {
	DeviceServiceMap map[string]string `json:"deviceservicemap"`
}

func verifyRTIOReq(req *RTIOReq) error {

	if req.Method != "getconfig" {
		log.Error().Err(ErrHTTPJSONInvalidMethod).Str("method", req.Method).Msg("method error")
		return ErrHTTPJSONInvalidMethod
	}
	return nil
}

func httpGetRTIOReq(r *http.Request) (*RTIOReq, error) {

	bodyBuf, err := io.ReadAll(r.Body)
	log.Debug().Int("bodylen", len(bodyBuf)).Msg("Get RTIOReq")
	if err != nil {
		return nil, ErrHTTPBodyReadFailed
	}

	req := &RTIOReq{}
	err = json.Unmarshal(bodyBuf, req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to  Unmarshal to RTIOReq")
		return nil, err
	}

	return req, nil
}
func httpWriteRTIOResp(w http.ResponseWriter, resp *RTIOResp) {
	buf, err := json.Marshal(*resp)
	if err != nil {
		log.Error().Err(err).Msg("Failed to Write RTIOResp, Marshal error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	len, err := w.Write(buf)
	if err != nil {
		log.Error().Err(err).Msg("Failed to Write RTIOResp, Write error")
		return
	}
	log.Debug().Int("datalen", len).Msg("Write RTIOResp")
}

func handler(w http.ResponseWriter, r *http.Request) {

	req, err := httpGetRTIOReq(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Error().Err(err).Msg("Failed to get RTIOReq")
		return
	}
	resp := &RTIOResp{
		ID:   req.ID,
		Code: RTIOCodeInternalServerError,
	}

	err = verifyRTIOReq(req)
	if err != nil {
		if err == ErrHTTPJSONInvalidMethod {
			resp.Code = RTIOCodeMethodNotAllowed
		}
		log.Error().Err(err).Msg("Failed to verify RTIOReq")
		httpWriteRTIOResp(w, resp)
		return
	}

	log.Info().Uint32("id", req.ID).Msg("handler")

	configText := `{
		"deviceservicemap": {
			"/aa/bb": "http://localhost:17517/deviceservice/aa/bb",
			"/aa/cc": "http://localhost:17517/deviceservice/aa/cc",
			"/aa/dd": "http://localhost:17518/deviceservice/aa/dd" 
		}  
	}`

	config := &Config{}
	err = json.Unmarshal([]byte(configText), config)

	if err != nil {
		log.Error().Err(err).Msg("Failed to Unmarshal")
		httpWriteRTIOResp(w, resp)
		return
	}

	configBuf, err := json.Marshal(config)
	if err != nil {
		log.Error().Err(err).Msg("Failed to Marshal")
		httpWriteRTIOResp(w, resp)
		return
	}

	resp.Digest = crc32.ChecksumIEEE(configBuf)
	resp.Config = string(configBuf)
	resp.Code = RTIOCodeOk

	log.Info().Uint32("id", req.ID).Uint32("digest", resp.Digest).Msg("handler")
	httpWriteRTIOResp(w, resp)
}

func main() {
	httpAddr := flag.String("http.addr", "0.0.0.0:17317", "address for http conntection")
	flag.Parse()

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05.000"}).With().Caller().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Info().Str("httpaddr", *httpAddr).Msg("hubconfiger started")
	log.Info().Str("url", "http://"+*httpAddr+"/hubconfiger").Msg("hubconfiger URL, replace the 0.0.0.0 address with the external address. ")

	http.HandleFunc("/hubconfiger", handler)

	err := http.ListenAndServe(*httpAddr, nil)
	if err != nil {
		log.Error().Err(err).Msg("hubconfiger failed")
	}
	log.Info().Str("httpaddr", *httpAddr).Msg("hubconfiger down")
}
