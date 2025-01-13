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
	"crypto"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	RTIODeviceIDLenMin = 30
	RTIODeviceIDLenMax = 40
	RTIOJWTTokenLenMin = 160
)

var (
	ErrHTTPBodyReadFailed           = errors.New("Read body failed")
	ErrHTTPBodyJsonUnmarshallFailed = errors.New("Read body json unmarshall failed")
	ErrHTTPJSONInvalidMethod        = errors.New("json req method error")
	ErrHTTPJSONInvalidDeviceID      = errors.New("json req deviceid error")
	ErrHTTPJSONInvalidExpires       = errors.New("json req expires error")
	ErrJWTTokenInvalid              = errors.New("JWT token invalid") // min token length, 36+124+0(ignore sign string)
)

const (
	RTIOCodeInternalServerError = "INTERNAL_SERVER_ERROR"
	RTIOCodeOk                  = "OK"
	RTIOCodeNotFound            = "NOT_FOUND"
	RTIOCodeBadRequest          = "BAD_REQUEST"
	RTIOCodeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
)

type RTIOReq struct {
	ID       uint32 `json:"id"`
	Method   string `json:"method"`
	DeviceID string `json:"deviceid"`
	Expires  uint32 `json:"expires"`
}

type RTIOResp struct {
	ID   uint32 `json:"id"`
	Code string `json:"code"`
	JWT  string `json:"jwt"`
}

type rtioHTTPHandler struct {
	ed25519PrivKey crypto.PrivateKey
}

func loadPrivKey(keyfile string) (crypto.PrivateKey, error) {
	key, err := os.ReadFile(keyfile)
	if err != nil {
		return nil, err
	}
	return jwt.ParseEdPrivateKeyFromPEM(key)
}

func verifyRTIOReq(req *RTIOReq) error {
	if req.Method != "jwtissuer" {
		log.Error().Err(ErrHTTPJSONInvalidMethod).Str("method", req.Method).Msg("method error")
		return ErrHTTPJSONInvalidMethod
	}
	if len(req.DeviceID) > 604800 {
		return ErrHTTPJSONInvalidDeviceID
	}
	if len(req.DeviceID) < RTIODeviceIDLenMin ||
		len(req.DeviceID) > RTIODeviceIDLenMax {
		return ErrHTTPJSONInvalidDeviceID
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

func (s *rtioHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req, err := httpGetRTIOReq(r)
	if err != nil {
		log.Error().Err(err).Msg("Failed to getJSONReq")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp := &RTIOResp{
		ID: req.ID,
	}
	err = verifyRTIOReq(req)
	if err != nil {
		if err == ErrHTTPJSONInvalidMethod {
			resp.Code = RTIOCodeMethodNotAllowed
		}
		log.Error().Err(err).Msg("Failed to verifyJSONReq")
		httpWriteRTIOResp(w, resp)
		return
	}
	if req.Expires == 0 {
		req.Expires = 604800
	}

	log.Info().Str("deviceid", req.DeviceID).Uint32("expires", req.Expires).Msg("Issue request")

	claims := jwt.NewWithClaims(&jwt.SigningMethodEd25519{},
		jwt.MapClaims{
			"iss": "rtio",
			"sub": req.DeviceID,
			"exp": time.Now().Unix() + int64(req.Expires),
		})

	token, err := claims.SignedString(s.ed25519PrivKey)
	if err != nil {
		log.Err(err).Msg("Fail to issue jwt")
		resp.Code = RTIOCodeInternalServerError
		httpWriteRTIOResp(w, resp)
		return
	}

	tokenLen := len(token)
	if tokenLen < RTIOJWTTokenLenMin {
		log.Err(ErrJWTTokenInvalid).Int("tokenlen", tokenLen).Msg("Issued")
		resp.Code = RTIOCodeInternalServerError
		httpWriteRTIOResp(w, resp)
		return
	}

	log.Info().Str("token", "*"+token[tokenLen-8:]).Msg("Issued")
	resp.JWT = token
	resp.Code = RTIOCodeOk
	httpWriteRTIOResp(w, resp)
}

func main() {

	httpAddr := flag.String("http.addr", "0.0.0.0:17019", "address for http conntection")
	ed25519PrivFile := flag.String("private.ed25519", "", "the private key (pem) for JWT")
	flag.Parse()

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05.000"}).With().Caller().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Info().Str("httpaddr", *httpAddr).Msg("jwtissuer started")

	if len(*ed25519PrivFile) == 0 {
		log.Error().Str("httpAddr", *httpAddr).Msg("jwtisuuer ed25519 private key empty")
		os.Exit(-1)
	}

	ed25519PrivKey, err := loadPrivKey(*ed25519PrivFile)
	if err != nil {
		log.Error().Err(err).Msg("jwtisuuer loadPrivKey")
		os.Exit(-1)
	}

	rtioHandler := &rtioHTTPHandler{
		ed25519PrivKey: ed25519PrivKey,
	}

	server := &http.Server{
		Addr:    *httpAddr,
		Handler: rtioHandler,
	}

	err = server.ListenAndServe()
	if err != nil {
		log.Error().Err(err).Msg("jwtissuer failed")
	}
	log.Info().Str("httpaddr", *httpAddr).Msg("jwtissuer down")

}
