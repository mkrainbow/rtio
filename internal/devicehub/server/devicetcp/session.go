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

package devicetcp

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mkrainbow/rtio/internal/devicehub/server/backendconn"
	"github.com/mkrainbow/rtio/internal/devicehub/server/service"
	"github.com/mkrainbow/rtio/pkg/config"
	dp "github.com/mkrainbow/rtio/pkg/deviceproto"
	"github.com/mkrainbow/rtio/pkg/rtioutil"
	ru "github.com/mkrainbow/rtio/pkg/rtioutil"
	"github.com/mkrainbow/rtio/pkg/timekv"

	"github.com/rs/zerolog/log"
)

const (
	HEARTBEAT_SECONDS_DEFAULT = 300
)

var (
	OutgoingChanSize = 10

	ErrRtioInternalServerError   = errors.New("ErrRtioInternalServerError")
	ErrSessionNotFound           = errors.New("ErrSessionNotFound")
	ErrDataType                  = errors.New("ErrDataType")
	ErrOverCapacity              = errors.New("ErrOverCapacity")
	ErrSendTimeout               = errors.New("ErrSendTimeout")
	ErrSendRespChannClose        = errors.New("ErrSendRespChannClose")
	ErrSessionVerifyData         = errors.New("ErrSessionVerifyData")
	ErrSessionVerifyNotCompleted = errors.New("ErrSessionVerifyNotCompleted")
	ErrSessionHeartbeatTimeout   = errors.New("ErrSessionHeartbeatTimeout")
	ErrHeaderIDNotExist          = errors.New("ErrHeaderIDNotExist")
	ErrObserverNotMatch          = errors.New("ErrObserverNotMatch")
	ErrObserverNotFound          = errors.New("ErrObserverNotFound")
	ErrMethodNotAllowed          = errors.New("ErrMethodNotAllowed")
	ErrResourceNotFound          = errors.New("ErrResourceNotFound")
	ErrMethodNotMatch            = errors.New("ErrMethodNotMatch")
)

type Message struct {
	ID   uint32
	Data []byte
}

type Session struct {
	deviceID              string
	conn                  net.Conn
	outgoingChan          chan []byte
	sendIDStore           *timekv.TimeKV
	observerStore         sync.Map
	rollingHeaderID       uint16 // rolling number for header id
	rollingHeaderIDLock   sync.Mutex
	rollingObserverID     uint16 // rolling number for observer id
	rollingObserverIDLock sync.Mutex
	observerCount         atomic.Int32
	BodyCapSize           uint16
	heartbeatSeconds      uint16
	RemoteAddr            net.Addr
	verifyPass            bool
	cancel                context.CancelFunc
	done                  chan struct{}
}

func newSession(conn net.Conn) *Session {
	s := &Session{
		conn:             conn,
		outgoingChan:     make(chan []byte, OutgoingChanSize),
		verifyPass:       false,
		done:             make(chan struct{}, 1),
		heartbeatSeconds: HEARTBEAT_SECONDS_DEFAULT,
	}
	s.sendIDStore = timekv.NewTimeKV(time.Second * 120)
	s.rollingHeaderID = 0
	s.rollingObserverID = 0
	s.observerCount.Store(0)
	return s
}

type Observa struct { // Observation
	ObserverID      uint16
	FrameID         uint32
	NotifyChan      chan *dp.ObGetNotifyReq
	SessionDoneChan chan struct{}
}

func calcuCheckSenconds(heartbeat uint16) time.Duration {
	return time.Duration(heartbeat + (heartbeat >> 1)) // heartbeat * 1.5
}

func (s *Session) genHeaderID() uint16 {
	s.rollingHeaderIDLock.Lock()
	s.rollingHeaderID++
	if s.rollingHeaderID == (uint16)(0) {
		s.rollingHeaderID = 1
	}
	s.rollingHeaderIDLock.Unlock()
	return s.rollingHeaderID
}
func (s *Session) genObserverID() uint16 {
	s.rollingObserverIDLock.Lock()
	s.rollingObserverID++
	if s.rollingObserverID == (uint16)(0) {
		s.rollingObserverID = 1
	}
	s.rollingObserverIDLock.Unlock()
	return s.rollingObserverID
}
func (s *Session) CreateObserva() (*Observa, error) {
	if s.observerCount.Load() >= dp.OBGET_OBSERVERS_MAX {
		return nil, dp.StatusCode_TooManyObservers
	}
	ob := &Observa{
		ObserverID:      s.genObserverID(),
		FrameID:         0,
		NotifyChan:      make(chan *dp.ObGetNotifyReq, 1),
		SessionDoneChan: make(chan struct{}, 1),
	}
	s.observerStore.Store(ob.ObserverID, ob)
	s.observerCount.Add(1)
	log.Debug().Uint16("obid", ob.ObserverID).Int32("obcount", s.observerCount.Load()).Msg("create observa")
	return ob, nil
}
func (s *Session) DestroyObserva(id uint16) {
	if v, ok := s.observerStore.LoadAndDelete(id); ok {
		ob := v.(*Observa)
		close(ob.NotifyChan)
		s.observerCount.Add(-1)
		log.Debug().Uint16("obid", ob.ObserverID).Int32("obcount", s.observerCount.Load()).Msg("destroy observa")
	}
}

func (s *Session) sendObEstabReq(ob *Observa, uri uint32, headerID uint16, data []byte) (<-chan []byte, error) {
	req := &dp.ObGetEstabReq{
		HeaderID: headerID,
		Method:   dp.Method_ObservedGet,
		ObID:     ob.ObserverID,
		URI:      uri,
		Data:     data,
	}

	buf := make([]byte, dp.HeaderLen+dp.HeaderLen_ObGetEstabReq+uint16(len(data)))
	if err := dp.EncodeObGetEstabReq_OverServerSendReq(req, buf); err != nil {
		log.Error().Uint16("obid", ob.ObserverID).Err(err).Msg("send ObEstabReq")
		return nil, err
	}

	respChan := make(chan []byte, 1)
	s.sendIDStore.Set(timekv.Key(headerID), &timekv.Value{C: respChan})
	s.outgoingChan <- buf
	return respChan, nil
}

func (s *Session) receiveObEstabResp(ob *Observa, headerID uint16, respChan <-chan []byte, timeout time.Duration) (dp.StatusCode, error) {

	defer s.sendIDStore.Del(timekv.Key(headerID))
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case sendRespBody, ok := <-respChan:
		if !ok {
			log.Error().Uint16("obid", ob.ObserverID).Err(ErrSendRespChannClose).Msg("receive ObEstabResp")
			return dp.StatusCode_Unknown, ErrSendRespChannClose
		}
		obEstabResp, err := dp.DecodeObGetEstabResp(headerID, sendRespBody)
		if err != nil {
			log.Error().Uint16("obid", ob.ObserverID).Err(err).Msg("receive ObEstabResp")
			return dp.StatusCode_Unknown, err
		}
		if obEstabResp.ObID != ob.ObserverID {
			log.Error().Uint16("obid", ob.ObserverID).Uint16("obid", ob.ObserverID).Uint16("resp.obid", obEstabResp.ObID).Err(ErrObserverNotMatch).Msg("receive ObEstabResp")
			return dp.StatusCode_Unknown, ErrObserverNotMatch
		}
		if obEstabResp.Method != dp.Method_ObservedGet {
			log.Error().Uint16("obid", ob.ObserverID).Err(dp.StatusCode_MethodNotAllowed).Msg("receive ObEstabResp")
			return dp.StatusCode_MethodNotAllowed, nil
		}
		return obEstabResp.Code, nil
	case <-t.C:
		log.Error().Uint16("obid", ob.ObserverID).Err(ErrSendTimeout).Msg("Send")
		return dp.StatusCode_Unknown, ErrSendTimeout
	}
}

func (s *Session) receiveObNotifyReq(header *dp.Header, buf []byte) error {
	req, err := dp.DecodeObGetNotifyReq(header.ID, buf)
	if err != nil {
		log.Error().Uint16("obid", req.ObID).Err(err).Msg("receive ObNotifyReq")
		return err
	}

	resp := &dp.ObGetNotifyResp{
		HeaderID: req.HeaderID,
		Method:   req.Method,
		ObID:     req.ObID,
	}
	if ob, ok := s.observerStore.Load(req.ObID); !ok {
		// observer ID not found, means observer not interesting, terminate observation
		resp.Code = dp.StatusCode_Terminate
		log.Info().Uint16("obid", req.ObID).Msg("receive ObNotifyReq, not found observa and terminate it")
	} else {
		resp.Code = dp.StatusCode_Continue
		ob.(*Observa).NotifyChan <- req
	}

	if err := s.sendObNotifyResp(resp); err != nil {
		log.Error().Uint16("obid", req.ObID).Err(err).Msg("receive ObNotifyReq")
	}
	return nil
}

func (s *Session) sendObNotifyResp(resp *dp.ObGetNotifyResp) error {
	log.Info().Uint16("obid", resp.ObID).Uint16("headerid", resp.HeaderID).Msg("send ObNotifyResp")
	buf := make([]byte, dp.HeaderLen+dp.HeaderLen_ObGetNotifyResp)
	if err := dp.EncodeObGetNotifyResp_OverDeviceSendResp(resp, buf); err != nil {
		log.Error().Uint16("obid", resp.ObID).Err(err).Msg("send ObNotifyResp")
		return err
	}
	s.outgoingChan <- buf
	return nil
}
func (s *Session) ObGetEstablish(ctx context.Context, uri uint32, ob *Observa, data []byte, timeout time.Duration) (dp.StatusCode, error) {

	headerID := s.genHeaderID()
	respChan, err := s.sendObEstabReq(ob, uri, headerID, data)
	if err != nil {
		return dp.StatusCode_Unknown, err
	}
	statusCode, err := s.receiveObEstabResp(ob, headerID, respChan, timeout)
	if err != nil {
		return dp.StatusCode_Unknown, err
	}
	return statusCode, nil
}

func (s *Session) sendCoReq(uri uint32, method dp.Method, headerID uint16, data []byte) (<-chan []byte, error) {
	req := &dp.CoReq{
		HeaderID: headerID,
		Method:   method,
		URI:      uri,
		Data:     data,
	}
	log.Info().Uint16("headerid", headerID).Uint32("uri", uri).Msg("send CoReq")
	buf := make([]byte, dp.HeaderLen+dp.HeaderLen_CoReq+uint16(len(data)))
	if err := dp.EncodeCoReq_OverServerSendReq(req, buf); err != nil {
		log.Error().Uint16("headerid", headerID).Err(err).Msg("send CoReq")
		return nil, err
	}
	respChan := make(chan []byte, 1)
	s.sendIDStore.Set(timekv.Key(headerID), &timekv.Value{C: respChan})
	s.outgoingChan <- buf
	return respChan, nil
}

func (s *Session) receiveCoResp(headerID uint16, respChan <-chan []byte, timeout time.Duration) (dp.StatusCode, []byte, error) {
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

func (s *Session) Send(ctx context.Context, uri uint32, method dp.Method, data []byte, timeout time.Duration) (dp.StatusCode, []byte, error) {
	headerID := s.genHeaderID()
	respChan, err := s.sendCoReq(uri, method, headerID, data)
	if err != nil {
		return dp.StatusCode_Unknown, nil, err
	}
	statusCode, data, err := s.receiveCoResp(headerID, respChan, timeout)
	if err != nil {
		return dp.StatusCode_Unknown, nil, err
	}
	return statusCode, data, nil
}

func (s *Session) Cancel() {
	if nil == s.cancel {
		log.Error().Msg("Session cancel is nil")
	}
	s.cancel()
}
func (s *Session) Done() <-chan struct{} {
	return s.done
}
func (s *Session) receiveVerifyReq(ctx context.Context, header *dp.Header) (bool, error) {
	reqBodyBuf := make([]byte, header.BodyLen)
	readLen, err := io.ReadFull(s.conn, reqBodyBuf)
	if err != nil {
		log.Error().Int("readLen", readLen).Err(err).Msg("Failed to read buf")
		return false, err
	}
	req, err := dp.DecodeVerifyReqBody(header, reqBodyBuf)
	if err != nil {
		log.Error().Err(err).Msg("Failed to decode req body")
		return false, err
	}
	if len(req.DeviceID) == 0 || len(req.DeviceSecret) == 0 {
		log.Error().Err(ErrSessionVerifyData).Msg("DeviceID or DeviceSecret Error")
		return false, s.sendVerifyResp(header, dp.Code_ParaInvalid)
	}
	// device verify
	if !config.BoolKV.GetWithDefault("disable.deviceverify", false) {

		verifyClient, err := backendconn.GetDeviceVerifier()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get device verify client")
			return false, s.sendVerifyResp(header, dp.Code_UnkownErr)
		}
		ok, err := verifyClient.Verify(req.DeviceID, req.DeviceSecret)
		if err != nil {
			log.Error().Err(err).Msg("call Verify err")
			err = s.sendVerifyResp(header, dp.Code_UnkownErr)
			return s.verifyPass, err
		}
		if !ok {
			log.Warn().Err(err).Str("deviceid", req.DeviceID).Msg("Validation Failed")
			err = s.sendVerifyResp(header, dp.Code_VerifyFail)
			return false, err
		}
	}
	// verify pass
	capSize, err := dp.GetCapSize(req.CapLevel)
	if err != nil {
		log.Error().Err(err).Msg("Failed to GetCapSize")
		err = s.sendVerifyResp(header, dp.Code_UnkownErr)
		return false, err
	}
	s.verifyPass = true
	s.BodyCapSize = capSize
	s.deviceID = req.DeviceID
	err = s.sendVerifyResp(header, dp.Code_Success)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Session) sendVerifyResp(header *dp.Header, code dp.RemoteCode) error {
	resp := &dp.VerifyResp{
		Header: &dp.Header{
			Version: dp.Version,
			Type:    dp.MsgType_DeviceVerifyResp,
			ID:      header.ID,
			BodyLen: 0,
			Code:    code,
		},
	}
	respBuf, err := dp.EncodeVerifyResp(resp)
	if err != nil {
		log.Error().Err(err).Msg("send VerifyResp")
		return err
	}
	s.outgoingChan <- respBuf
	return nil
}

func (s *Session) devicePingProcess(header *dp.Header) error {

	bodyBuf := make([]byte, header.BodyLen)
	readLen, err := io.ReadFull(s.conn, bodyBuf)
	if err != nil {
		log.Error().Int("readLen", readLen).Err(err).Msg("device ping, ReadFull")
		return err
	}

	respCode := dp.Code_Success
	req, err := dp.DecodePingReqBody(header, bodyBuf)
	if err != nil {
		log.Error().Err(err).Msg("device ping, decode ping request")
		if err == dp.ErrLengthError {
			respCode = dp.Code_LengthErr
		} else {
			return err
		}
	} else {
		// update timeout value by device
		if req.Timeout != 0 {
			if req.Timeout < 30 || req.Timeout > 43200 { // 43200 secodes = 12 hours
				respCode = dp.Code_ParaInvalid
			} else {
				s.heartbeatSeconds = req.Timeout
			}
		}
	}
	// send ping resp
	resp := &dp.PingResp{
		Header: &dp.Header{
			Version: dp.Version,
			Type:    dp.MsgType_DevicePingResp,
			ID:      header.ID,
			BodyLen: 0,
			Code:    respCode,
		},
	}
	respBuf, err := dp.EncodePingResp(resp)
	if err != nil {
		return err
	}
	s.outgoingChan <- respBuf
	return nil
}
func (s *Session) serverSendRespone(header *dp.Header) error {
	value, ok := s.sendIDStore.Get(timekv.Key(header.ID))

	if !ok {
		log.Warn().Uint16("headerid", header.ID).Err(ErrHeaderIDNotExist).Msg("handle server Respone")
		return ErrHeaderIDNotExist
	}

	bodyBuf := make([]byte, header.BodyLen)
	readLen, err := io.ReadFull(s.conn, bodyBuf)
	if err != nil {
		log.Error().Int("readLen", readLen).Err(err).Msg("handle server Respone, ReadFull")
		return err
	}

	sendResp, err := dp.DecodeSendRespBody(header, bodyBuf)
	if err != nil {
		log.Error().Err(err).Msg("handle server Respone, decode body")
		return err
	}
	value.C <- sendResp.Body
	return nil
}

func (s *Session) receiveCoReq(header *dp.Header, buf []byte) error {
	req, err := dp.DecodeCoReq(header.ID, buf)
	if err != nil {
		log.Error().Err(err).Msg("receive CoReq")
		return err
	}
	resp := &dp.CoResp{
		HeaderID: req.HeaderID,
		Method:   req.Method,
	}

	url, ok := config.StringKV.Get("deviceservice." + strconv.FormatUint(uint64(req.URI), 16))

	if ok {
		c, err := backendconn.GetServiceClient()
		if err == nil {
			id, err := rtioutil.GenUint32ID()
			if err == nil {
				log.Info().Uint16("headerid", req.HeaderID).Uint32("postid", id).Uint32("uri", req.URI).Msg("Post to device service")
				data, err := c.Post(id, url, s.deviceID, req.Data)
				if err == nil {
					resp.Data = data
					resp.Code = dp.StatusCode_OK
				} else {
					if err == service.ErrBadRequest {
						resp.Code = dp.StatusCode_BadRequest
					} else {
						resp.Code = dp.StatusCode_InternalServerError
					}
				}
			} else {
				log.Error().Err(err).Msg("Failed to get uint32 id")
			}
		} else {
			log.Error().Err(err).Msg("Falied to get client for service")
			resp.Code = dp.StatusCode_InternalServerError
		}

	} else {
		resp.Code = dp.StatusCode_NotFount
	}

	err = s.sendCoResp(resp)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send resp to device")
	}
	return nil
}
func (s *Session) sendCoResp(resp *dp.CoResp) error {
	log.Info().Uint16("headerid", resp.HeaderID).Str("status", resp.Code.String()).Msg("send CoResp")
	buf := make([]byte, dp.HeaderLen+dp.HeaderLen_CoResp+uint16(len(resp.Data)))
	if err := dp.EncodeCoResp_OverDeviceSendResp(resp, buf); err != nil {
		log.Error().Err(err).Msg("send CoResp")
		return err
	}
	s.outgoingChan <- buf
	return nil
}

func (s *Session) deviceSendRequest(header *dp.Header) error {

	bodyBuf := make([]byte, header.BodyLen)
	readLen, err := io.ReadFull(s.conn, bodyBuf)
	if err != nil {
		log.Error().Int("readLen", readLen).Err(err).Msg("handle device request, ReadFull")
		return err
	}

	method, err := dp.DecodeMethod(bodyBuf)
	if err != nil {
		log.Error().Err(err).Msg("handle device request")
		return err
	}

	switch method {
	case dp.Method_ConstrainedGet, dp.Method_ConstrainedPost:
		err := s.receiveCoReq(header, bodyBuf)
		if err != nil {
			log.Error().Err(err).Msg("handle device request")
			return err
		}
	case dp.Method_ObservedGet:
		err := s.receiveObNotifyReq(header, bodyBuf)
		if err != nil {
			log.Error().Err(err).Msg("handle device request")
			return err
		}
	default:
		log.Error().Err(ErrMethodNotAllowed).Msg("handle device request")
	}
	return nil
}
func (s *Session) tcpIncomming(serveCtx context.Context, addSession func(context.Context, string, *Session),
	verifyTimer *time.Timer, heartbeatTimer *time.Ticker, errChan chan<- error) {
	defer func() {
		log.Debug().Msg("Incomming route exit")
	}()

	for {
		select {
		case <-serveCtx.Done():
			log.Debug().Msg("Incomming route context done")
			return
		default:

			headBuf := make([]byte, dp.HeaderLen)
			readLen, err := io.ReadFull(s.conn, headBuf)
			log.Debug().Int("readlen", readLen).Hex("buf", headBuf).Msg("Incomming route, read header")
			if err != nil {
				errChan <- err
				return
			}
			header, err := dp.DecodeHeader(headBuf)
			if err != nil {
				errChan <- dp.ErrDecode
				return
			}

			switch header.Type {
			case dp.MsgType_DeviceVerifyReq:
				if ok, err := s.receiveVerifyReq(serveCtx, header); err != nil {
					errChan <- err
					return
				} else {
					if ok { // verify passed
						addSession(serveCtx, s.deviceID, s)
						verifyTimer.Stop()
						log.Debug().Msg("Incomming route, verify passed, verifyTimer stoped")
					}
				}
			case dp.MsgType_DevicePingReq:
				if err := s.devicePingProcess(header); err != nil {
					errChan <- err
					return
				}
				log.Debug().Uint16("heartbeat", s.heartbeatSeconds).Msg("Incomming route, ping req")
				heartbeatTimer.Reset(time.Second * calcuCheckSenconds(s.heartbeatSeconds))
			case dp.MsgType_DeviceSendReq:
				if err := s.deviceSendRequest(header); err != nil {
					errChan <- err
					return
				}
				heartbeatTimer.Reset(time.Second * calcuCheckSenconds(s.heartbeatSeconds))
			case dp.MsgType_ServerSendResp:
				if err := s.serverSendRespone(header); err != nil {
					errChan <- err
					return
				}
				heartbeatTimer.Reset(time.Second * calcuCheckSenconds(s.heartbeatSeconds))
			default:
				log.Error().Err(ErrDataType).Uint8("type", uint8(header.Type)).Msg("Incomming route")
				errChan <- ErrDataType
				return
			}

		}
	}
}
func (s *Session) tcpOutgoing(serveCtx context.Context, errChan chan<- error) {
	defer func() {
		log.Debug().Msg("Outgoing route exit")
	}()

	for {
		select {

		case <-serveCtx.Done():
			log.Debug().Msg("Outgoing route  context done")
			return
		case buf := <-s.outgoingChan:
			if n, err := ru.WriteFull(s.conn, buf); err != nil {
				log.Error().Err(err).Int("writelen", n).Int("buflen", len(buf)).Msg("Outgoing route  WriteFull error")
				errChan <- err
				return
			}
		}
	}
}
func (s *Session) serve(ctx context.Context, wait *sync.WaitGroup,
	addSession func(context.Context, string, *Session),
	delSession func(string)) {
	var serveCtx context.Context
	serveCtx, s.cancel = context.WithCancel(ctx)
	serveLogger := log.With().Str("client_ip", s.conn.RemoteAddr().String()).Logger()
	serveLogger.Info().Msg("start serving")
	defer func() {
		serveLogger.Info().Msg("stop service for this conn")
	}()

	s.RemoteAddr = s.conn.RemoteAddr()

	defer wait.Done()
	defer s.conn.Close()
	defer func() {
		// close all observations
		s.observerStore.Range(func(k, v any) bool {
			log.Info().Uint16("obid", k.(uint16)).Msg("send done observation")
			v.(*Observa).SessionDoneChan <- struct{}{}
			return true
		})
		// delete session when verify pass and session be created
		if s.verifyPass {
			delSession(s.deviceID)
		}
		s.done <- struct{}{}
	}()

	verifyTimer := time.NewTimer(time.Second * 15)
	defer verifyTimer.Stop()
	heartbeatTimer := time.NewTicker(time.Second * calcuCheckSenconds(s.heartbeatSeconds))
	defer heartbeatTimer.Stop()
	errChan := make(chan error, 2)
	go s.tcpOutgoing(serveCtx, errChan)
	go s.tcpIncomming(serveCtx, addSession, verifyTimer, heartbeatTimer, errChan)

	storeTicker := time.NewTicker(time.Second * 5)
	defer storeTicker.Stop()

	for {
		select {
		case <-verifyTimer.C:
			log.Debug().Err(ErrSessionVerifyNotCompleted).Msg("verifyTimer timeout")
			errChan <- ErrSessionVerifyNotCompleted
			return
		case <-heartbeatTimer.C:
			log.Debug().Err(ErrSessionHeartbeatTimeout).Msg("Incomming route heartbeatTimer timeout")
			errChan <- ErrSessionHeartbeatTimeout
			return
		case err := <-errChan:
			s.cancel()
			log.Warn().Err(err).Msg("serve done when error")
			return
		case <-serveCtx.Done():
			log.Debug().Msg("serve done when ctx done")
			if err := s.conn.Close(); err != nil {
				log.Debug().Err(err).Msg("sever close conn error")
			}
			return
		case <-storeTicker.C:
			// servelogger.Debug().Msgf("storeTicker.C %s", time.Now().String())
			s.sendIDStore.DelExpireKeys()
		}
	}
}
