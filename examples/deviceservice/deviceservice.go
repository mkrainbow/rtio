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
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	RTIOCodeInternalServerError = "INTERNAL_SERVER_ERROR"
	RTIOCodeOk                  = "OK"
	RTIOCodeBadRequest          = "BAD_REQUEST"
	RTIOCodeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
)

var (
	ErrHTTPBodyReadFailed           = errors.New("Read body failed")
	ErrHTTPBodyJsonUnmarshallFailed = errors.New("Read body json unmarshall failed")
	ErrHTTPJSONInvalidData          = errors.New("json req Data error")
	ErrHTTPJSONInvalidBinaryData    = errors.New("json req Binary Data error")
	ErrHTTPJSONInvalidMethod        = errors.New("json req method error")
)

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

func verifyRTIOReq(req *RTIOReq) error {

	if req.Method != "copost" {
		log.Error().Err(ErrHTTPJSONInvalidMethod).Str("method", req.Method).Msg("verifyJSONReq")
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

func handlerBB(w http.ResponseWriter, r *http.Request) {

	req, err := httpGetRTIOReq(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Error().Err(err).Msg("Failed to get req")
		return
	}
	resp := &RTIOResp{
		ID:   req.ID,
		Code: RTIOCodeInternalServerError,
	}

	err = verifyRTIOReq(req)
	if err != nil {
		resp.Code = RTIOCodeBadRequest
		log.Error().Err(err).Msg("Failed to verify req")
		httpWriteRTIOResp(w, resp)
		return
	}

	data, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		resp.Code = RTIOCodeBadRequest
		log.Error().Err(err).Msg("Failed to decode req")
		httpWriteRTIOResp(w, resp)
		return
	}

	log.Info().Uint32("id", req.ID).Str("deviceid", req.DeviceID).Str("data", string(data)).Msg("req data")
	buf := []byte("deviceservice: respone with bb")
	resp.Code = RTIOCodeOk
	resp.Data = base64.StdEncoding.EncodeToString(buf)
	log.Info().Uint32("id", req.ID).Str("deviceid", req.DeviceID).Str("data", string(buf)).Msg("resp data")

	httpWriteRTIOResp(w, resp)
}

func handlerCC(w http.ResponseWriter, r *http.Request) {

	req, err := httpGetRTIOReq(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Error().Err(err).Msg("Failed to get req")
		return
	}
	resp := &RTIOResp{
		ID:   req.ID,
		Code: RTIOCodeInternalServerError,
	}

	err = verifyRTIOReq(req)
	if err != nil {
		resp.Code = RTIOCodeBadRequest
		log.Error().Err(err).Msg("Failed to verify req")
		httpWriteRTIOResp(w, resp)
		return
	}

	data, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		resp.Code = RTIOCodeBadRequest
		log.Error().Err(err).Msg("Failed to decode req")
		httpWriteRTIOResp(w, resp)
		return
	}

	log.Info().Uint32("id", req.ID).Str("deviceid", req.DeviceID).Str("data", string(data)).Msg("req data")
	buf := []byte("deviceservice: respone with cc")
	resp.Code = RTIOCodeOk
	resp.Data = base64.StdEncoding.EncodeToString(buf)
	log.Info().Uint32("id", req.ID).Str("deviceid", req.DeviceID).Str("data", string(buf)).Msg("resp data")

	httpWriteRTIOResp(w, resp)
}

func handlerDD(w http.ResponseWriter, r *http.Request) {

	req, err := httpGetRTIOReq(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Error().Err(err).Msg("Failed to get req")
		return
	}
	resp := &RTIOResp{
		ID:   req.ID,
		Code: RTIOCodeInternalServerError,
	}

	err = verifyRTIOReq(req)
	if err != nil {
		resp.Code = RTIOCodeBadRequest
		log.Error().Err(err).Msg("Failed to verify req")
		httpWriteRTIOResp(w, resp)
		return
	}

	data, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		resp.Code = RTIOCodeBadRequest
		log.Error().Err(err).Msg("Failed to decode req")
		httpWriteRTIOResp(w, resp)
		return
	}

	log.Info().Uint32("id", req.ID).Str("deviceid", req.DeviceID).Str("data", string(data)).Msg("req data")

	buf := []byte("deviceservice: respone with dd")
	resp.Code = RTIOCodeOk
	resp.Data = base64.StdEncoding.EncodeToString(buf)
	log.Info().Uint32("id", req.ID).Str("deviceid", req.DeviceID).Str("data", string(buf)).Msg("resp data")

	httpWriteRTIOResp(w, resp)
}

func main() {
	httpAddr := flag.String("http.addr", "0.0.0.0:17517", "address for http conntection")
	flag.Parse()

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05.000"}).With().Caller().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Info().Str("httpaddr", *httpAddr).Msg("deviceserice started")

	http.HandleFunc("/deviceservice/aa/bb", handlerBB)
	http.HandleFunc("/deviceservice/aa/cc", handlerCC)
	// http.HandleFunc("/deviceservice/aa/dd", handlerDD)

	err := http.ListenAndServe(*httpAddr, nil)
	if err != nil {
		log.Error().Err(err).Msg("deviceservice failed")
	}
	log.Info().Str("httpaddr", *httpAddr).Msg("deviceservice down")
}
