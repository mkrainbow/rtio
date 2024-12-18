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
	"context"
	"crypto"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"sync"

	"github.com/mkrainbow/rtio/pkg/config"
	"github.com/mkrainbow/rtio/pkg/rpcproto/devicehub"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var (
	ErrHTTPMissingAuthorizationHeader = errors.New("Missing or malformed Authorization header")
	ErrHTTPInvalidAuthorizationHeader = errors.New("Invalid Authorization header")
	ErrHTTPInvalidJWT                 = errors.New("Invalid or expired JWT")
	ErrHTTPDeviceIDLen                = errors.New("The deviceID length error")
	ErrHTTPBodyLenExceedsLimit        = errors.New("The body length exceeds the limit")
	ErrHTTPBodyReadFailed             = errors.New("Read body failed")
	ErrHTTPBodyJsonUnmarshallFailed   = errors.New("Read body json unmarshall failed")
	ErrHTTPJSONInvalidURI             = errors.New("json req URI error")
	ErrHTTPJSONInvalidData            = errors.New("json req Data error")
	ErrHTTPJSONInvalidBinaryData      = errors.New("json req Binary Data error")
	ErrHTTPJSONInvalidMethod          = errors.New("json req method error")

	ErrJWTPubKeyEmpty      = errors.New("JWT keyfile is empty")
	ErrJWTPubKeyLoadFailed = errors.New("JWT keyfile is empty")
	ErrJWTTokenInvalid     = errors.New("JWT token invalid") // min token length, 36+124+0(ignore sign string)
)

const (
	RTIOHttpBodyLenMax         = 864 // URILenMax(128) + Base64DataLenMax  + other(64) = 832
	RTIODeviceIDLenMin         = 30
	RTIODeviceIDLenMax         = 40
	RTIODeviceURILenMin        = 4
	RTIODeviceURILenMax        = 128
	RTIOHttpBase64DataLenMax   = 672 // DateLenMax*4/3 = 672
	RTIODeviceBinaryDateLenMax = 504 // 512-8 (remain for REST-Like Protocol)
	RTIOJWTTokenLenMin         = 160
)

const (
	RTIOCodeInternalServerError = "INTERNAL_SERVER_ERROR"
	RTIOCodeOk                  = "OK"
	RTIOCodeDeviceOffline       = "DEVICEID_OFFLINE"
	RTIOCodeDeviceTimeout       = "DEVICEID_TIMEOUT"
	RTIOCodeContinue            = "CONTINUE"
	RTIOCodeTerminate           = "TERMINATE"
	RTIOCodeNotFound            = "NOT_FOUND"
	RTIOCodeBadRequest          = "BAD_REQUEST"
	RTIOCodeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
	RTIOCodeTooManyRequests     = "TOO_MANY_REQUESTS"
	RTIOCodeTooManyObservers    = "TOO_MANY_OBSERVERS"
	RTIOCodeRequestTimeout      = "REQUEST_TIMEOUT"
)

type RTIOReq struct {
	ID     uint32 `json:"id"`
	Method string `json:"method"`
	URI    string `json:"uri"`
	Data   string `json:"data"`
}

type RTIOResp struct {
	ID      uint32 `json:"id"`
	FrameID uint32 `json:"fid"` // option
	Code    string `json:"code"`
	Data    string `json:"data"`
}

type rtioHTTPHandler struct {
	hub       devicehub.AccessServiceClient
	jwtPubKey crypto.PublicKey
}

func transHubCode(code devicehub.Code) string {

	switch code {
	case devicehub.Code_CODE_INTERNAL_SERVER_ERROR:
		return RTIOCodeInternalServerError
	case devicehub.Code_CODE_OK:
		return RTIOCodeOk
	case devicehub.Code_CODE_DEVICEID_OFFLINE:
		return RTIOCodeDeviceOffline
	case devicehub.Code_CODE_DEVICEID_TIMEOUT:
		return RTIOCodeDeviceTimeout
	case devicehub.Code_CODE_CONTINUE:
		return RTIOCodeContinue
	case devicehub.Code_CODE_TERMINATE:
		return RTIOCodeTerminate
	case devicehub.Code_CODE_NOT_FOUNT:
		return RTIOCodeNotFound
	case devicehub.Code_CODE_BAD_REQUEST:
		return RTIOCodeBadRequest
	case devicehub.Code_CODE_METHOD_NOT_ALLOWED:
		return RTIOCodeMethodNotAllowed
	case devicehub.Code_CODE_TOO_MANY_REQUESTS:
		return RTIOCodeTooManyRequests
	case devicehub.Code_CODE_TOO_MANY_OBSERVERS:
		return RTIOCodeTooManyObservers
	case devicehub.Code_CODE_REQUEST_TIMEOUT:
		return RTIOCodeRequestTimeout
	default:
		return RTIOCodeInternalServerError
	}
}

func loadPubKey(keyfile string) (crypto.PublicKey, error) {
	key, err := os.ReadFile(keyfile)
	if err != nil {
		return nil, err
	}

	return jwt.ParseEdPublicKeyFromPEM(key)
}

func getBinaryData(req *RTIOReq) ([]byte, error) {

	data, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil || len(data) > RTIODeviceBinaryDateLenMax {
		log.Error().Err(ErrHTTPJSONInvalidBinaryData).Int("datalen", len(data)).Msg("Failed to get binary data")
		return nil, ErrHTTPJSONInvalidBinaryData
	}
	return data, nil
}

func verifyRTIOReq(req *RTIOReq) error {

	if req.Method != "copost" && req.Method != "obget" {
		log.Warn().Err(ErrHTTPJSONInvalidMethod).Str("method", req.Method).Msg("Failed to verify RTIOReq")
		return ErrHTTPJSONInvalidMethod
	}
	uriLen := len(req.URI)
	if (uriLen < RTIODeviceURILenMin) || (uriLen > RTIODeviceURILenMax) {
		log.Warn().Err(ErrHTTPJSONInvalidURI).Int("urilen", uriLen).Msg("Failed to verify RTIOReq, uri length error")
		return ErrHTTPJSONInvalidURI
	}
	dataLen := len(req.Data)
	if dataLen > RTIOHttpBase64DataLenMax {
		log.Warn().Err(ErrHTTPJSONInvalidData).Int("dataLen", dataLen).Msg("Failed to verify RTIOReq, base64 length error")
		return ErrHTTPJSONInvalidData
	}
	return nil
}
func httpGetDeviceID(r *http.Request) (string, error) {
	log.Debug().Str("path", r.URL.Path).Msg("Get device id")
	seg := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(seg) < 1 ||
		len(seg[0]) < RTIODeviceIDLenMin ||
		len(seg[0]) > RTIODeviceIDLenMax {
		return "", ErrHTTPDeviceIDLen
	}
	return seg[0], nil
}

func httpGetRTIOReq(r *http.Request) (*RTIOReq, error) {

	limitReader := io.LimitReader(r.Body, RTIOHttpBodyLenMax+1) // 1 more for bound
	bodyBuf, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, ErrHTTPBodyReadFailed
	}
	log.Debug().Int("bodylen", len(bodyBuf)).Msg("Get RTIOReq from http.Request")
	if len(bodyBuf) > RTIOHttpBodyLenMax {
		log.Warn().Int("bodylen", len(bodyBuf)).Msg("Get RTIOReq bodylen exceed limit")
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
func httpWriteRTIORespStream(w http.ResponseWriter, f http.Flusher, resp *RTIOResp) {
	buf, err := json.Marshal(*resp)
	if err != nil {
		log.Error().Err(err).Msg("Failed to Write RTIOResp with stream, Marshal error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	buf = append(buf, byte('\n'))
	len, err := w.Write(buf)
	if err != nil {
		log.Error().Err(err).Msg("Failed to Write RTIOResp with stream, Write error")
		return
	}
	f.Flush()
	log.Debug().Int("datalen", len).Msg("Write RTIOResp with Stream")
}
func (s *rtioHTTPHandler) validateJWT(token string) (string, error) {
	tokenLen := len(token)
	if tokenLen < RTIOJWTTokenLenMin {
		log.Err(ErrJWTTokenInvalid).Int("tokenlen", tokenLen).Msg("Failed to validate JWT")
		return "", ErrJWTTokenInvalid
	}
	log.Debug().Str("token", "*"+token[tokenLen-8:]).Msg("Failed to validate JWT")

	t, err := jwt.Parse(string(token), func(t *jwt.Token) (interface{}, error) {
		return s.jwtPubKey, nil
	}, jwt.WithValidMethods([]string{"EdDSA"}), // only suppored ed25519
		jwt.WithLeeway(time.Duration(10)*time.Second)) // with 10s leeway

	if err != nil {
		log.Err(err).Msg("Failed to validate JWT")
		return "", err
	}

	sub, err := t.Claims.GetSubject()
	if err != nil {
		log.Err(err).Msg("Failed to validate JWT")
		return "", err
	}
	return sub, nil
}

func (s *rtioHTTPHandler) validateToken(r *http.Request) (string, error) {

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrHTTPMissingAuthorizationHeader
	}

	const prefix = "Bearer "
	if len(authHeader) < len(prefix) || authHeader[:len(prefix)] != prefix {
		return "", ErrHTTPInvalidAuthorizationHeader
	}
	tokenString := authHeader[len(prefix):]

	sub, err := s.validateJWT(tokenString)
	if err != nil {
		return "", ErrHTTPInvalidJWT
	}
	return sub, nil
}
func (s *rtioHTTPHandler) serveCoPost(w http.ResponseWriter, r *http.Request,
	deviceID string, rtioReq *RTIOReq, rtioResp *RTIOResp) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	data, err := getBinaryData(rtioReq)
	if err != nil {
		rtioResp.Code = RTIOCodeBadRequest
		log.Error().Err(err).Msg("Fail to copost")
		httpWriteRTIOResp(w, rtioResp)
		return
	}

	req := &devicehub.CoReq{
		DeviceId: deviceID,
		Uri:      rtioReq.URI,
		Id:       uint32(rtioReq.ID),
		Data:     data,
	}

	resp, err := s.hub.CoPost(r.Context(), req)
	if err != nil {
		rtioResp.Code = RTIOCodeInternalServerError
		log.Error().Err(err).Msg("Fail to copost, device hub error")
		httpWriteRTIOResp(w, rtioResp)
		return
	}
	rtioResp.Code = transHubCode(resp.Code)
	if resp.Code == devicehub.Code_CODE_OK && len(resp.Data) > 0 {
		rtioResp.Data = base64.StdEncoding.EncodeToString(resp.Data)
	}
	httpWriteRTIOResp(w, rtioResp)
}

func (s *rtioHTTPHandler) serveObGet(w http.ResponseWriter, r *http.Request,
	deviceID string, rtioReq *RTIOReq, rtioResp *RTIOResp) {

	f, ok := w.(http.Flusher)
	if !ok {
		log.Error().Msg("http client Streaming unsupported")
		http.Error(w, "Streaming unsupported!", http.StatusNotAcceptable)
		return
	}
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	data, err := getBinaryData(rtioReq)
	if err != nil {
		rtioResp.Code = RTIOCodeBadRequest
		log.Error().Err(err).Msg("Fail to obget")
		httpWriteRTIOResp(w, rtioResp)
		return
	}

	req := &devicehub.ObGetReq{
		DeviceId: deviceID,
		Uri:      rtioReq.URI,
		Id:       uint32(rtioReq.ID),
		Data:     data,
	}

	respStream, err := s.hub.ObGet(r.Context(), req)
	if err != nil {
		rtioResp.Code = RTIOCodeInternalServerError
		log.Error().Err(err).Msg("Fail to obget, device hub error")
		httpWriteRTIOResp(w, rtioResp)
		return
	}

	for {
		resp, err := respStream.Recv()
		if err == io.EOF {
			return
		} else if err != nil {
			grpcStatus, isGrpcStatus := status.FromError(err)
			if isGrpcStatus {
				if codes.Canceled == grpcStatus.Code() {
					log.Warn().Msg("Canceled for http request context")
					return
				}
			}
			rtioResp.Code = RTIOCodeInternalServerError
			log.Error().Err(err).Msg("Fail to obget, device hub error")
			httpWriteRTIORespStream(w, f, rtioResp)
			return
		}
		rtioResp.Code = transHubCode(resp.Code)
		rtioResp.FrameID = resp.Fid
		rtioResp.Data = base64.StdEncoding.EncodeToString(resp.Data)
		httpWriteRTIORespStream(w, f, rtioResp)
	}
}

func (s *rtioHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	log.Debug().Msg("handle rtio http reqest")
	deviceID, err := httpGetDeviceID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		log.Error().Err(err).Msg("Failed to get deviceID")
		return
	}
	log.Info().Str("deviceID", deviceID).Msg("handle rtio http reqest")

	if config.BoolKV.GetWithDefault("enable.jwt", false) {
		subject, err := s.validateToken(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			log.Warn().Err(err).Msg("handle rtio http reqest, Failed to verify token")
			return
		}
		if subject != deviceID {
			http.Error(w, "JWT subject invalid", http.StatusUnauthorized)
			log.Warn().Str("subject", subject).Str("deviceid", deviceID).Err(err).Msg("handle rtio http reqest, subject not match")
			return
		}
	}

	rtioReq, err := httpGetRTIOReq(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Warn().Err(err).Msg("Failed to get RTIOReq")
		return
	}

	rtioResp := &RTIOResp{
		ID:   rtioReq.ID,
		Code: RTIOCodeInternalServerError,
	}

	err = verifyRTIOReq(rtioReq)
	if err != nil {
		if err == ErrHTTPJSONInvalidMethod {
			rtioResp.Code = RTIOCodeMethodNotAllowed
		} else if err == ErrHTTPJSONInvalidData || err == ErrHTTPJSONInvalidURI {
			rtioResp.Code = RTIOCodeBadRequest
		}
		log.Warn().Err(err).Msg("Failed to verify RTIOReq")
		httpWriteRTIOResp(w, rtioResp)
		return
	}

	if rtioReq.Method == "copost" {
		s.serveCoPost(w, r, deviceID, rtioReq, rtioResp)
	} else if rtioReq.Method == "obget" {
		s.serveObGet(w, r, deviceID, rtioReq, rtioResp)
	}

}

func InitHttpsGateway(ctx context.Context, rpcAddr, gwAddr string,
	wait *sync.WaitGroup,
	certFile, keyFile string) error {

	conn, err := grpc.NewClient(
		rpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		log.Error().Err(err).Msg("Failed to dial server")
		return err
	}
	log.Info().Str("rpcaddr", rpcAddr).Msg("connected")

	rtioHandler := &rtioHTTPHandler{
		hub: devicehub.NewAccessServiceClient(conn),
	}

	if config.BoolKV.GetWithDefault("enable.jwt", false) {
		keyPem := config.StringKV.GetWithDefault("jwt.ed25519", "")
		if keyPem == "" {
			log.Error().Err(ErrJWTPubKeyEmpty).Msg("jwt ed25519 public key file empty")
			return ErrJWTPubKeyEmpty
		}
		rtioHandler.jwtPubKey, err = loadPubKey(keyPem)
		if err != nil {
			log.Error().Err(ErrJWTPubKeyLoadFailed).Msg("jwt ed25519 public key load failed")
			return ErrJWTPubKeyLoadFailed
		}
	}

	if certFile == "" || keyFile == "" {
		log.Error().Msg("TLS certfile or keyfile is empty")
		return errors.New("TLS certfile or keyfile is empty")
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load key pair")
		return errors.New("Failed to load key pair")
	}
	gwServer := &http.Server{
		Addr:      gwAddr,
		Handler:   rtioHandler,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	log.Info().Str("gwaddr", gwAddr).Msg("gateway started with TLS")
	wait.Add(1)
	go func() {
		defer wait.Done()
		err = gwServer.ListenAndServeTLS(certFile, keyFile)
		if err != nil {
			if err == http.ErrServerClosed {
				log.Info().Msg("gateway http closed")
				return
			}
			log.Error().Err(err).Msg("gateway serve failed")
		}
	}()

	go func() {
		<-ctx.Done()
		log.Info().Msg("gateway ctx down")
		gwServer.Shutdown(ctx)
	}()

	return nil
}

func InitHttpGateway(ctx context.Context, rpcAddr, gwAddr string, wait *sync.WaitGroup) error {

	conn, err := grpc.NewClient(
		rpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		log.Error().Err(err).Msg("Failed to dial server")
		return err
	}
	log.Info().Str("rpcaddr", rpcAddr).Msg("connected")

	rtioHandler := &rtioHTTPHandler{
		hub: devicehub.NewAccessServiceClient(conn),
	}

	if config.BoolKV.GetWithDefault("enable.jwt", false) {
		keyPem := config.StringKV.GetWithDefault("jwt.ed25519", "")
		if keyPem == "" {
			log.Error().Err(ErrJWTPubKeyEmpty).Msg("jwt ed25519 public key file empty")
			return ErrJWTPubKeyEmpty
		}
		rtioHandler.jwtPubKey, err = loadPubKey(keyPem)
		if err != nil {
			log.Error().Err(ErrJWTPubKeyLoadFailed).Msg("jwt ed25519 public key load failed")
			return ErrJWTPubKeyLoadFailed
		}
	}

	gwServer := &http.Server{
		Addr:    gwAddr,
		Handler: rtioHandler,
	}
	log.Info().Str("gwaddr", gwAddr).Msg("gateway started")
	wait.Add(1)
	go func() {
		defer wait.Done()
		err = gwServer.ListenAndServe()
		if err != nil {
			if err == http.ErrServerClosed {
				log.Info().Msg("gateway http closed")
				return
			}
			log.Error().Err(err).Msg("gateway serve failed")
		}
	}()

	go func() {
		<-ctx.Done()
		log.Info().Msg("gateway ctx down")
		gwServer.Shutdown(ctx)
	}()

	return nil
}
