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

package main

import (
	"encoding/json"
	"errors"
	"flag"
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
	RTIOCodeVerificationFailed  = "VERIFICATION_FAILED"
)

type RTIOReq struct {
	ID           int    `json:"id"`
	Method       string `json:"method"`
	DeviceID     string `json:"deviceid"`
	DeviceSecret string `json:"devicesecret"`
}

type RTIOResp struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
}

func verifyRTIOReq(req *RTIOReq) error {

	if req.Method != "verify" {
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
		log.Error().Err(err).Msg("handler, getJSONReq")
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
		log.Error().Err(err).Msg("handler, verifyJSONReq")
		httpWriteRTIOResp(w, resp)
		return
	}

	log.Info().Int("id", req.ID).Str("deviceid", req.DeviceID).Msg("handler")

	if (req.DeviceID == "cfa09baa-4913-4ad7-a936-2e26f9671b05" && req.DeviceSecret == "mb6bgso4EChvyzA05thF9+wH") ||
		(req.DeviceID == "cfa09baa-4913-4ad7-a936-3e26f9671b10" && req.DeviceSecret == "mb6bgso4EChvyzA05thF9+He") ||
		(req.DeviceID == "cfa09baa-4913-4ad7-a936-3e26f9671b09" && req.DeviceSecret == "mb6bgso4EChvyzA05thF9+wH") {
		resp.Code = RTIOCodeOk
	} else {
		resp.Code = RTIOCodeVerificationFailed
	}
	httpWriteRTIOResp(w, resp)
}

// curl test this example
// $ curl http://localhost:17217/deviceverifier -d '{"method":"verify","id": 1999,"deviceid":"cfa09baa-4913-4ad7-a936-2e26f9671b05", "devicesecret": "mb6bgso4EChvyzA05thF9+wH"}'
// {"id":1999,"code":"OK"}

func main() {
	httpAddr := flag.String("http.addr", "0.0.0.0:17217", "address for http connection")
	flag.Parse()

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05.000"}).With().Caller().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Info().Str("httpaddr", *httpAddr).Msg("deviceverifier started")

	log.Info().Str("url", "http://"+*httpAddr+"/deviceverifier").Msg("deviceverifier URL, replace the 0.0.0.0 address with the external address. ")

	http.HandleFunc("/deviceverifier", handler)

	err := http.ListenAndServe(*httpAddr, nil)
	if err != nil {
		log.Error().Err(err).Msg("deviceverifier failed")
	}
	log.Info().Str("httpaddr", *httpAddr).Msg("deviceverifier down")
}
