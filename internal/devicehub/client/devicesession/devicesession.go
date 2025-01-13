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

package devicesession

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	dp "github.com/mkrainbow/rtio/pkg/deviceproto"
	ru "github.com/mkrainbow/rtio/pkg/rtioutil"
	"github.com/mkrainbow/rtio/pkg/timekv"

	"github.com/rs/zerolog/log"
)

// Default Settings for connection.
const (
	DeviceHeartbeatSecondsDefault = 300
	DeviceConnectMaxTimes         = 20
	DeviceConnectIntervalSeconds  = 3
)

// Error Code for the Connection.
var (
	ErrDataType             = errors.New("ErrDataType")
	ErrOverCapacity         = errors.New("ErrOverCapacity")
	ErrSendTimeout          = errors.New("ErrSendTimeout")
	ErrCanceled             = errors.New("ErrCanceled")
	ErrVerifyFailed         = errors.New("ErrVerifyFailed")
	ErrPingFailed           = errors.New("ErrPingFailed")
	ErrTransFunc            = errors.New("ErrTransFunc")
	ErrObserverNotMatch     = errors.New("ErrObserverNotMatch")
	ErrMethodNotMatch       = errors.New("ErrMethodNotMatch")
	ErrSendRespChannClose   = errors.New("ErrSendRespChannClose")
	ErrHeaderIDNotExist     = errors.New("ErrHeaderIDNotExist")
	ErrConnectTimesExceeded = errors.New("ErrConnectTimesExceeded")
)

// ConnectOptions holds the options for establishing a connection to a server.
type ConnnectOptions struct {
	method    string
	localAddr string
	caFile    string
}

// DeviceSession represents a session for interacting with a connected device.
type DeviceSession struct {
	deviceID            string
	deviceSecret        string
	serverAddr          string
	connectOptions      *ConnnectOptions
	outgoingChan        chan []byte
	errChan             chan error
	regPostHandlerMap   map[uint32]func(req []byte) ([]byte, error)
	regObGetHandlerMap  map[uint32]func(ctx context.Context, req []byte) (<-chan []byte, error)
	sendIDStore         *timekv.TimeKV
	rollingHeaderID     uint16
	rollingHeaderIDLock sync.Mutex
	conn                net.Conn
	heartbeatSeconds    uint16
	reconnectTimes      uint16
}

// Connect establishes a connection to a server with the provided device credentials.
func Connect(ctx context.Context, deviceID, deviceSecret, serverAddr string) (*DeviceSession, error) {
	var conn net.Conn
	var err error
	tryTime := 1
	for ; tryTime <= DeviceConnectMaxTimes; tryTime++ {
		conn, err = net.DialTimeout("tcp", serverAddr, time.Second*60)
		if err != nil {
			log.Error().Err(err).Int("trytimes", tryTime).Msg("Connect server error, connect later")
			time.Sleep(time.Second * DeviceConnectIntervalSeconds)
			continue
		}
		break
	}
	if tryTime > DeviceConnectMaxTimes {
		log.Error().Err(ErrConnectTimesExceeded).Int("time", tryTime).Msg("Connect server error, retry times exceeded")
		return nil, ErrConnectTimesExceeded
	}
	session := newDeviceSession(conn, deviceID, deviceSecret, serverAddr, nil)
	return session, nil
}

// ConnectWithLocalAddr establishes a connection with a specified local address. Usually for testing.
func ConnectWithLocalAddr(ctx context.Context, deviceID, deviceSecret, localAddr, serverAddr string) (*DeviceSession, error) {

	connOptions := &ConnnectOptions{method: "with_local_addr", localAddr: localAddr}

	laddr, err := net.ResolveTCPAddr("tcp", localAddr)
	if err != nil {
		log.Error().Err(err).Msg("localAddr error")
		return nil, err
	}
	dialer := net.Dialer{LocalAddr: laddr, Timeout: time.Second * 60}

	var conn net.Conn
	tryTime := 1
	for ; tryTime <= DeviceConnectMaxTimes; tryTime++ {
		conn, err = dialer.Dial("tcp", serverAddr)
		if err != nil {
			log.Error().Err(err).Int("trytimes", tryTime).Msg("Connect server error, connect later")
			time.Sleep(time.Second * DeviceConnectIntervalSeconds)
			continue
		}
		break
	}
	if tryTime > DeviceConnectMaxTimes {
		log.Error().Err(ErrConnectTimesExceeded).Int("time", tryTime).Msg("Connect server error, retry times exceeded")
		return nil, ErrConnectTimesExceeded
	}

	session := newDeviceSession(conn, deviceID, deviceSecret, serverAddr, connOptions)
	return session, nil
}

// ConnectWithTLS establishes a TLS-encrypted connection to the server.
func ConnectWithTLS(ctx context.Context, deviceID, deviceSecret, serverAddr, caFile string) (*DeviceSession, error) {

	connOptions := &ConnnectOptions{method: "with_tls", caFile: caFile}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		log.Error().Err(err).Msg("read CA file error")
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		log.Error().Msg("append CA cert error")
		return nil, fmt.Errorf("failed to append CA cert")
	}
	conf := &tls.Config{
		RootCAs: caCertPool,
	}
	dialer := &net.Dialer{Timeout: time.Second * 60}

	var conn net.Conn
	tryTime := 1
	for ; tryTime <= DeviceConnectMaxTimes; tryTime++ {
		conn, err = tls.DialWithDialer(dialer, "tcp", serverAddr, conf)
		if err != nil {
			log.Error().Err(err).Int("trytimes", tryTime).Msg("Connect server error, connect later")
			time.Sleep(time.Second * DeviceConnectIntervalSeconds)
			continue
		}
		break
	}
	if tryTime > DeviceConnectMaxTimes {
		log.Error().Err(ErrConnectTimesExceeded).Int("time", tryTime).Msg("Connect server error, retry times exceeded")
		return nil, ErrConnectTimesExceeded
	}

	session := newDeviceSession(conn, deviceID, deviceSecret, serverAddr, connOptions)
	return session, nil
}

// ConnectWithTLSSkipVerify establishes a TLS-encrypted connection, skipping certificate verification.
func ConnectWithTLSSkipVerify(ctx context.Context, deviceID, deviceSecret, serverAddr string) (*DeviceSession, error) {

	connOptions := &ConnnectOptions{method: "with_tls_skip_verify"}

	conf := &tls.Config{
		InsecureSkipVerify: true,
	}
	dialer := &net.Dialer{Timeout: time.Second * 60}

	var err error
	var conn net.Conn
	tryTime := 1
	for ; tryTime <= DeviceConnectMaxTimes; tryTime++ {
		conn, err = tls.DialWithDialer(dialer, "tcp", serverAddr, conf)
		if err != nil {
			log.Error().Err(err).Int("trytime", tryTime).Msg("Connect server error, connect later")
			time.Sleep(time.Second * DeviceConnectIntervalSeconds)
			continue
		}
		break
	}
	if tryTime > DeviceConnectMaxTimes {
		log.Error().Err(ErrConnectTimesExceeded).Int("time", tryTime).Msg("Connect server error, retry times exceeded")
		return nil, ErrConnectTimesExceeded
	}

	session := newDeviceSession(conn, deviceID, deviceSecret, serverAddr, connOptions)
	return session, nil
}

// SetHeartbeatSeconds sets the heartbeat interval in seconds for the device session.
func (s *DeviceSession) SetHeartbeatSeconds(n uint16) {
	s.heartbeatSeconds = n
}

// Serve starts the device session and serves requests in background until the context is canceled.
func (s *DeviceSession) Serve(ctx context.Context) {
	go s.serve(ctx, s.errChan)
	go s.recover(ctx, s.errChan)
}
func newDeviceSession(conn net.Conn, deviceID, deviceSecret, serverAddr string, connectOptions *ConnnectOptions) *DeviceSession {
	s := &DeviceSession{
		deviceID:           deviceID,
		deviceSecret:       deviceSecret,
		serverAddr:         serverAddr,
		connectOptions:     connectOptions,
		outgoingChan:       make(chan []byte, 10),
		errChan:            make(chan error, 1),
		sendIDStore:        timekv.NewTimeKV(time.Second * 120),
		regPostHandlerMap:  make(map[uint32]func(req []byte) ([]byte, error), 1),
		regObGetHandlerMap: make(map[uint32]func(ctx context.Context, req []byte) (<-chan []byte, error), 1),
		conn:               conn,
		heartbeatSeconds:   DeviceHeartbeatSecondsDefault,
		reconnectTimes:     0,
	}
	s.rollingHeaderID = 0

	return s
}
func (s *DeviceSession) genHeaderID() uint16 {
	s.rollingHeaderIDLock.Lock()
	s.rollingHeaderID++
	if s.rollingHeaderID == (uint16)(0) {
		s.rollingHeaderID = 1
	}
	s.rollingHeaderIDLock.Unlock()
	return s.rollingHeaderID
}

func (s *DeviceSession) verify() error {
	headerID, err := ru.GenUint16ID()
	if err != nil {
		return err
	}
	req := &dp.VerifydReq{
		Header: &dp.Header{
			Version: dp.Version,
			Type:    dp.MsgType_DeviceVerifyReq,
			ID:      headerID,
		},
		CapLevel:     1,
		DeviceID:     s.deviceID,
		DeviceSecret: s.deviceSecret,
	}

	buf, err := dp.EncodeVerifyReq(req)
	if err != nil {
		return err
	}
	s.outgoingChan <- buf
	return nil
}

// ping heartbeat: 0 - not update, other - using heartbeat as new value
func (s *DeviceSession) ping() error {
	headerID, err := ru.GenUint16ID()
	if err != nil {
		return err
	}
	req := &dp.PingReq{
		Header: &dp.Header{
			Version: dp.Version,
			Type:    dp.MsgType_DevicePingReq,
			ID:      headerID,
		},
		Timeout: 0,
	}

	buf, err := dp.EncodePingReq(req)
	if err != nil {
		return err
	}
	s.outgoingChan <- buf
	return nil
}

// pingRemoteSet timeout [30, 43200]
func (s *DeviceSession) pingRemoteSet(timeout uint16) error {
	headerID, err := ru.GenUint16ID()
	if err != nil {
		return err
	}
	req := &dp.PingReq{
		Header: &dp.Header{
			Version: dp.Version,
			Type:    dp.MsgType_DevicePingReq,
			ID:      headerID,
		},
		Timeout: timeout,
	}

	buf, err := dp.EncodePingReq(req)
	if err != nil {
		return err
	}
	s.outgoingChan <- buf
	return nil
}

// getReconnectInterval
// interval : [2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768 ...] secodes
// level    :  0, 1, 2,  3,  4,  5,   6,   7,   8,    9,   10,   11,   12,    13,    14
// recommend: set maxlevel=8 (max retry intervel is 8.5 minite)
func (s *DeviceSession) getReconnectInterval(maxLevel uint16) time.Duration {
	n := maxLevel
	if s.reconnectTimes < n {
		n = s.reconnectTimes
	}
	return 2 << n
}
func (s *DeviceSession) reconnect(ctx context.Context, serverAddr string, errChan chan<- error) uint16 {
	s.reconnectTimes++

	var conn net.Conn
	var err error
	if s.connectOptions == nil {
		conn, err = net.DialTimeout("tcp", serverAddr, time.Second*60)
		if err != nil {
			log.Error().Str("deviceid", s.deviceID).Uint16("retrytimes", s.reconnectTimes).Err(err).Msg("Reconnect error")
			errChan <- err
			return s.reconnectTimes
		}
	} else if s.connectOptions.method == "with_local_addr" {
		laddr, err := net.ResolveTCPAddr("tcp", s.connectOptions.localAddr)
		if err != nil {
			log.Error().Str("deviceid", s.deviceID).Uint16("retrytimes", s.reconnectTimes).Err(err).Msg("Reconnect with local addr, addr resolve error")
			errChan <- err
			return s.reconnectTimes
		}
		dialer := net.Dialer{LocalAddr: laddr, Timeout: time.Second * 60}
		conn, err = dialer.Dial("tcp", serverAddr)
		if err != nil {
			log.Error().Str("deviceid", s.deviceID).Uint16("retrytimes", s.reconnectTimes).Err(err).Msg("Reconnect with local addr, dial error")
			errChan <- err
			return s.reconnectTimes
		}
	} else if s.connectOptions.method == "with_tls" {
		caCert, err := os.ReadFile(s.connectOptions.caFile)
		if err != nil {
			log.Error().Str("deviceid", s.deviceID).Uint16("retrytimes", s.reconnectTimes).Err(err).Msg("Reconnect with tls, ca read error")
			errChan <- err
			return s.reconnectTimes
		}
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			log.Error().Str("deviceid", s.deviceID).Uint16("retrytimes", s.reconnectTimes).Err(err).Msg("Reconnect with tls, ca append error")
			errChan <- err
			return s.reconnectTimes
		}
		conf := &tls.Config{
			RootCAs: caCertPool,
		}
		dialer := &net.Dialer{Timeout: time.Second * 60}
		conn, err = tls.DialWithDialer(dialer, "tcp", serverAddr, conf)
		if err != nil {
			log.Error().Str("deviceid", s.deviceID).Uint16("retrytimes", s.reconnectTimes).Err(err).Msg("Reconnect with tls, dail error")
			errChan <- err
			return s.reconnectTimes
		}
	} else if s.connectOptions.method == "with_tls_skip_verify" {
		conf := &tls.Config{
			InsecureSkipVerify: true,
		}
		dialer := &net.Dialer{Timeout: time.Second * 60}
		conn, err = tls.DialWithDialer(dialer, "tcp", serverAddr, conf)
		if err != nil {
			log.Error().Str("deviceid", s.deviceID).Uint16("retrytimes", s.reconnectTimes).Err(err).Msg("Reconnect with tls skip verify, dail error")
			errChan <- err
			return s.reconnectTimes
		}
	}

	s.reconnectTimes = 0 // connect sucess and reset to 0
	s.conn = conn
	log.Info().Str("deviceid", s.deviceID).Uint16("retrytimes", s.reconnectTimes).Msg("Reconnect server success")
	go s.serve(ctx, errChan)
	return s.reconnectTimes
}

func (s *DeviceSession) sendCoReq(headerID uint16, method dp.Method, uri uint32, data []byte) (<-chan []byte, error) {
	req := &dp.CoReq{
		HeaderID: headerID,
		Method:   method,
		URI:      uri,
		Data:     data,
	}
	log.Info().Uint16("headerid", headerID).Uint32("uri", uri).Msg("send CoReq")
	buf := make([]byte, int(dp.HeaderLen+dp.HeaderLen_CoReq)+len(data))
	if err := dp.EncodeCoReq_OverDeviceSendReq(req, buf); err != nil {
		log.Error().Uint16("headerid", headerID).Err(err).Msg("send CoReq")
		return nil, err
	}
	respChan := make(chan []byte, 1)
	s.sendIDStore.Set(timekv.Key(headerID), &timekv.Value{C: respChan})
	s.outgoingChan <- buf
	return respChan, nil
}
func (s *DeviceSession) receiveCoResp(headerID uint16, respChan <-chan []byte, timeout time.Duration) (dp.StatusCode, []byte, error) {
	defer s.sendIDStore.Del(timekv.Key(headerID))
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case sendRespBody, ok := <-respChan:
		if !ok {
			log.Error().Err(ErrSendRespChannClose).Msg("receive CoResp")
			return dp.StatusCode_Unknown, nil, ErrSendRespChannClose
		}
		coResp, err := dp.DecodeCoResp(headerID, sendRespBody)
		if err != nil {
			log.Error().Err(err).Msg("receive CoResp")
			return dp.StatusCode_Unknown, nil, err
		}
		if coResp.Method != dp.Method_ConstrainedGet && coResp.Method != dp.Method_ConstrainedPost {
			log.Error().Err(ErrMethodNotMatch).Msg("receive CoResp")
			return dp.StatusCode_InternalServerError, nil, ErrMethodNotMatch
		}
		log.Info().Uint16("headerid", coResp.HeaderID).Str("status", coResp.Code.String()).Msg("receive CoResp")

		return coResp.Code, coResp.Data, nil
	case <-t.C:
		log.Error().Err(ErrSendTimeout).Msg("receive CoResp")
		return dp.StatusCode_Unknown, nil, ErrSendTimeout
	}
}

func (s *DeviceSession) receiveCoRespWithContext(ctx context.Context, headerID uint16, respChan <-chan []byte) (dp.StatusCode, []byte, error) {
	defer s.sendIDStore.Del(timekv.Key(headerID))
	select {
	case sendRespBody, ok := <-respChan:
		if !ok {
			log.Error().Err(ErrSendRespChannClose).Msg("receive CoResp")
			return dp.StatusCode_Unknown, nil, ErrSendRespChannClose
		}
		coResp, err := dp.DecodeCoResp(headerID, sendRespBody)
		if err != nil {
			log.Error().Err(err).Msg("receive CoResp")
			return dp.StatusCode_Unknown, nil, err
		}
		if coResp.Method != dp.Method_ConstrainedGet && coResp.Method != dp.Method_ConstrainedPost {
			log.Error().Err(ErrMethodNotMatch).Msg("receive CoResp")
			return dp.StatusCode_InternalServerError, nil, ErrMethodNotMatch
		}
		log.Info().Uint16("headerid", coResp.HeaderID).Str("status", coResp.Code.String()).Msg("receive CoResp")

		return coResp.Code, coResp.Data, nil
	case <-ctx.Done():
		log.Info().Msg("context done")
		return dp.StatusCode_Unknown, nil, ErrCanceled
	}
}

func (s *DeviceSession) sendObNotifyReq(obid uint16, headerID uint16, data []byte) (<-chan []byte, error) {
	req := &dp.ObGetNotifyReq{
		HeaderID: headerID,
		ObID:     obid,
		Method:   dp.Method_ObservedGet,
		Code:     dp.StatusCode_Continue,
		Data:     data,
	}
	log.Info().Uint16("obid", obid).Uint16("headerid", headerID).Msg("send ObNotifyReq")
	buf := make([]byte, int(dp.HeaderLen+dp.HeaderLen_ObGetNotifyReq)+len(data))
	if err := dp.EncodeObGetNotifyReq_OverDeviceSendReq(req, buf); err != nil {
		return nil, err
	}
	respChan := make(chan []byte, 1)
	s.sendIDStore.Set(timekv.Key(headerID), &timekv.Value{C: respChan})
	s.outgoingChan <- buf
	return respChan, nil
}
func (s *DeviceSession) sendObNotifyReqTerminate(obid uint16, headerID uint16) (<-chan []byte, error) {
	req := &dp.ObGetNotifyReq{
		HeaderID: headerID,
		ObID:     obid,
		Method:   dp.Method_ObservedGet,
		Code:     dp.StatusCode_Terminate,
	}
	log.Info().Uint16("obid", obid).Uint16("headerid", headerID).Msg("send ObNotifyReq wtih Terminate")
	buf := make([]byte, int(dp.HeaderLen+dp.HeaderLen_ObGetNotifyReq))
	if err := dp.EncodeObGetNotifyReq_OverDeviceSendReq(req, buf); err != nil {
		return nil, err
	}
	respChan := make(chan []byte, 1)
	s.sendIDStore.Set(timekv.Key(headerID), &timekv.Value{C: respChan})
	s.outgoingChan <- buf
	return respChan, nil
}
func (s *DeviceSession) receiveObNotifyResp(obID, headerID uint16, respChan <-chan []byte, timeout time.Duration) (dp.StatusCode, error) {
	defer s.sendIDStore.Del(timekv.Key(headerID))
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case sendRespBody, ok := <-respChan:
		if !ok {
			log.Error().Uint16("obid", obID).Err(ErrSendRespChannClose).Msg("receive ObNotifyResp")
			return dp.StatusCode_Unknown, ErrSendRespChannClose
		}
		obEstabResp, err := dp.DecodeObGetNotifyResp(headerID, sendRespBody)
		if err != nil {
			log.Error().Uint16("obid", obID).Err(err).Msg("receive ObNotifyResp")
			return dp.StatusCode_Unknown, err
		}
		if obEstabResp.ObID != obID {
			log.Error().Uint16("obid", obID).Uint16("resp.obid", obEstabResp.ObID).Err(ErrObserverNotMatch).Msg("receive ObNotifyResp")
			return dp.StatusCode_Unknown, ErrObserverNotMatch
		}
		if obEstabResp.Method != dp.Method_ObservedGet {
			log.Error().Uint16("obid", obID).Err(dp.StatusCode_MethodNotAllowed).Msg("receive ObNotifyResp")
			return dp.StatusCode_MethodNotAllowed, nil
		}
		return obEstabResp.Code, nil
	case <-t.C:
		log.Error().Uint16("obid", obID).Err(ErrSendTimeout).Msg("receive ObNotifyResp")
		return dp.StatusCode_Unknown, ErrSendTimeout
	}
}
func (s *DeviceSession) observe(obID uint16, cancal context.CancelFunc, notifyReqChan <-chan []byte) {
	log.Info().Uint16("obid", obID).Msg("observe")

	defer func() {
		log.Info().Uint16("obid", obID).Msg("observe exit")
	}()
	defer cancal()

	for {
		data, ok := <-notifyReqChan
		headerID := s.genHeaderID()
		if ok {
			notfyRespChan, err := s.sendObNotifyReq(obID, headerID, data)
			if err != nil {
				log.Error().Uint16("obid", obID).Err(err).Msg("send ObNotifyReq error")
				return
			}
			statusCode, err := s.receiveObNotifyResp(obID, headerID, notfyRespChan, time.Second*20)
			if err != nil {
				log.Error().Uint16("obid", obID).Err(err).Msg("receive ObNotifyResp error")
				return
			}
			if dp.StatusCode_Continue != statusCode {
				log.Debug().Uint16("obid", obID).Str("status", statusCode.String()).Msg("observe terminiate")
				return
			}
		} else {
			log.Debug().Uint16("obid", obID).Msg("observe, notify req channel closed")
			notfyRespChan, err := s.sendObNotifyReqTerminate(obID, headerID)
			if err != nil {
				log.Error().Uint16("obid", obID).Err(err).Msg("send ObNotifyReq with Terminate error")
				return
			}
			_, err = s.receiveObNotifyResp(obID, headerID, notfyRespChan, time.Second*20)
			if err != nil {
				log.Error().Uint16("obid", obID).Err(err).Msg("receive ObNotifyResp error")
				return
			}
			return
		}
	}
}

func (s *DeviceSession) obGetReqHandler(header *dp.Header, reqBuf []byte) error {
	req, err := dp.DecodeObGetEstabReq(header.ID, reqBuf)
	if err != nil {
		log.Error().Uint16("obid", req.ObID).Err(err).Msg("obGet")
		return err
	}

	resp := &dp.ObGetEstabResp{
		HeaderID: req.HeaderID,
		Method:   req.Method,
		ObID:     req.ObID,
	}
	handler, ok := s.regObGetHandlerMap[req.URI]
	respBuf := make([]byte, dp.HeaderLen+dp.HeaderLen_ObGetEstabResp)
	if !ok {
		log.Warn().Uint16("obid", req.ObID).Uint32("uri", req.URI).Msg("obGet, uri not found")
		resp.Code = dp.StatusCode_NotFount
		if err := dp.EncodeObGetEstabResp_OverServerSendResp(resp, respBuf); err != nil {
			log.Error().Uint16("obid", req.ObID).Err(err).Msg("obGet")
			return err
		}
	} else {
		ctx, cancle := context.WithCancel(context.Background())
		respChan, err := handler(ctx, req.Data)
		if err != nil {
			resp.Code = dp.StatusCode_InternalServerError
			cancle()
			return err
		}
		resp.Code = dp.StatusCode_Continue
		if err := dp.EncodeObGetEstabResp_OverServerSendResp(resp, respBuf); err != nil {
			log.Error().Uint16("obid", req.ObID).Err(err).Msg("obGet")
			cancle()
			return err
		}
		go s.observe(req.ObID, cancle, respChan)
	}
	s.outgoingChan <- respBuf
	return nil
}

func (s *DeviceSession) coPostReqHandler(header *dp.Header, reqBuf []byte) error {
	req, err := dp.DecodeCoReq(header.ID, reqBuf)
	if err != nil {
		log.Error().Err(err).Msg("coPost")
		return err
	}
	resp := &dp.CoResp{
		HeaderID: req.HeaderID,
		Method:   req.Method,
	}
	handler, ok := s.regPostHandlerMap[req.URI]

	if !ok {
		log.Warn().Uint16("headerid", req.HeaderID).Uint32("uri", req.URI).Msg("CoPost, uri not found")
		resp.Code = dp.StatusCode_NotFount
		buf := make([]byte, int(dp.HeaderLen+dp.HeaderLen_CoResp))
		if err := dp.EncodeCoResp_OverServerSendResp(resp, buf); err != nil {
			log.Error().Uint16("headerid", req.HeaderID).Err(err).Msg("coPost")
			return err
		}
		s.outgoingChan <- buf
	} else {
		data, err := handler(req.Data)
		if err != nil {
			resp.Code = dp.StatusCode_InternalServerError
			return err
		}
		resp.Code = dp.StatusCode_OK
		resp.Data = data
		buf := make([]byte, int(dp.HeaderLen+dp.HeaderLen_CoResp)+len(resp.Data))
		if err := dp.EncodeCoResp_OverServerSendResp(resp, buf); err != nil {
			log.Error().Uint16("headerid", req.HeaderID).Err(err).Msg("oPost")
			return err
		}
		s.outgoingChan <- buf
	}
	return nil
}

func (s *DeviceSession) serverSendRequest(header *dp.Header) error {

	bodyBuf := make([]byte, header.BodyLen)
	readLen, err := io.ReadFull(s.conn, bodyBuf)
	if err != nil {
		log.Error().Int("readLen", readLen).Err(err).Msg("handle server request, ReadFull")
		return err
	}

	method, err := dp.DecodeMethod(bodyBuf)
	if err != nil {
		log.Error().Err(err).Msg("handle server request")
		return err
	}

	switch method {
	case dp.Method_ObservedGet:
		err := s.obGetReqHandler(header, bodyBuf)
		if err != nil {
			log.Error().Err(err).Msg("handle server request")
			return err
		}
	case dp.Method_ConstrainedPost:
		err := s.coPostReqHandler(header, bodyBuf)
		if err != nil {
			log.Error().Err(err).Msg("handle server request")
			return err
		}
	}
	return nil
}

func (s *DeviceSession) deviceSendRespone(header *dp.Header) error {
	value, ok := s.sendIDStore.Get(timekv.Key(header.ID))

	if !ok {
		log.Warn().Uint16("headerid", header.ID).Msg("handle device response")
		return ErrHeaderIDNotExist
	}
	bodyBuf := make([]byte, header.BodyLen)
	readLen, err := io.ReadFull(s.conn, bodyBuf)
	if err != nil {
		log.Error().Int("readLen", readLen).Err(err).Msg("handle device response, ReadFull")
		return err
	}

	sendResp, err := dp.DecodeSendRespBody(header, bodyBuf)
	if err != nil {
		log.Error().Err(err).Msg("handle device response")
		return err
	}
	value.C <- sendResp.Body
	return nil
}

func (s *DeviceSession) tcpIncomming(ctx context.Context, errChan chan<- error) {
	defer func() {
		log.Debug().Msg("Incomming route exit")
	}()

	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("Incomming route context done")
			return
		default:
			headerBuf := make([]byte, dp.HeaderLen)
			readLen, err := io.ReadFull(s.conn, headerBuf)
			log.Debug().Int("readlen", readLen).Msg("Incomming route, read header")
			if err != nil {
				errChan <- err
				return
			}
			header, err := dp.DecodeHeader(headerBuf)
			if err != nil {
				errChan <- dp.ErrDecode
				return
			}
			log.Debug().Uint16("headerid", header.ID).Str("type", header.Type.String()).Msg("Incomming route")
			switch header.Type {
			case dp.MsgType_DeviceVerifyResp:
				if header.Code == dp.Code_Success {
					log.Info().Msg("verify pass")
				} else {
					log.Error().Uint8("resp.code", uint8(header.Code)).Msg("verify fail")
					errChan <- ErrVerifyFailed
					return
				}
			case dp.MsgType_DevicePingResp:
				if header.Code == dp.Code_Success {
					log.Debug().Msg("ping success")
				} else {
					log.Error().Uint8("resp.code", uint8(header.Code)).Msg("ping fail")
					errChan <- ErrPingFailed
					return
				}
			case dp.MsgType_ServerSendReq:
				err := s.serverSendRequest(header)
				if err != nil {
					log.Error().Err(err).Msg("Incomming route")
					errChan <- err
					return
				}
			case dp.MsgType_DeviceSendResp:
				if err := s.deviceSendRespone(header); err != nil {
					errChan <- err
					return
				}
			default:
				errChan <- ErrDataType
				return
			}

		}
	}
}

func (s *DeviceSession) tcpOutgoing(ctx context.Context, heartbeat *time.Ticker, errChan chan<- error) {
	defer func() {
		log.Debug().Msg("Outgoing route exit")
	}()

	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("Outgoing route context done")
			return
		case buf := <-s.outgoingChan:
			heartbeat.Reset(time.Second * time.Duration(s.heartbeatSeconds))
			if n, err := ru.WriteFull(s.conn, buf); err != nil {
				log.Error().Err(err).Int("writelen", n).Int("buflen", len(buf)).Msg("Outgoing route WriteFull error")
				errChan <- err
				return
			}
		}
	}
}

func (s *DeviceSession) serve(ctx context.Context, errChan chan<- error) {

	devicelogger := log.With().Str("device_ip", s.conn.LocalAddr().String()).Logger()
	devicelogger.Info().Str("deviceid", s.deviceID).Msg("serving")
	defer func() {
		devicelogger.Info().Msg("stop serving")
	}()

	ioCtx, ioCancel := context.WithCancel(ctx)
	defer ioCancel()

	heartbeat := time.NewTicker(time.Second * time.Duration(s.heartbeatSeconds))
	defer heartbeat.Stop()

	errNetChan := make(chan error, 2)
	go s.tcpIncomming(ioCtx, errNetChan)
	go s.tcpOutgoing(ioCtx, heartbeat, errNetChan)

	if err := s.verify(); err != nil {
		log.Error().Err(err).Str("deviceip", s.conn.LocalAddr().String()).Msg("send verify error")
		return
	}

	// device define heartbeat's timeout
	s.heartbeatSeconds = 60
	if s.heartbeatSeconds != DeviceHeartbeatSecondsDefault {
		s.pingRemoteSet(s.heartbeatSeconds)
	}

	storeTicker := time.NewTicker(time.Second * 5)
	defer storeTicker.Stop()

	for {
		select {
		case err := <-errNetChan:
			errChan <- err
			log.Error().Err(err).Msg("device done when error")
			s.conn.Close()
			ioCancel()
			return
		case <-ioCtx.Done():
			log.Debug().Msg("device done when deviceCtx done")
			s.conn.Close()
			// if err := s.conn.Close(); err != nil {
			// 	log.Warn().Err(err).Msg("device close conn error")
			// }
			return
		case <-heartbeat.C:
			devicelogger.Debug().Msgf("ping req")
			s.ping()
		case <-storeTicker.C:
			s.sendIDStore.DelExpireKeys()
		}
	}
}

func (s *DeviceSession) recover(ctx context.Context, errChan chan error) {
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("recover context done")
			return
		case err := <-errChan:
			log.Error().Err(err).Msg("reconnect later for recover")
			time.AfterFunc(time.Second*s.getReconnectInterval(6), func() {
				s.reconnect(ctx, s.serverAddr, errChan)
			})
		}
	}
}
