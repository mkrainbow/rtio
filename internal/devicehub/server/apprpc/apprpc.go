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

package apprpc

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/mkrainbow/rtio/internal/devicehub/server/devicetcp"
	"github.com/mkrainbow/rtio/pkg/deviceproto"
	dp "github.com/mkrainbow/rtio/pkg/deviceproto"
	"github.com/mkrainbow/rtio/pkg/rpcproto/devicehub"
	"github.com/mkrainbow/rtio/pkg/rtioutil"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type AccessServer struct {
	devicehub.UnimplementedAccessServiceServer
	sessions *devicetcp.SessionMap
}

var (
	ErrNotFoundDevice = errors.New("ErrNotFoundDevice")
)

func transToRPCCode(code dp.StatusCode) devicehub.Code {
	switch code {
	case dp.StatusCode_Unknown:
		return devicehub.Code_CODE_INTERNAL_SERVER_ERROR
	case dp.StatusCode_OK:
		return devicehub.Code_CODE_OK
	case dp.StatusCode_Continue:
		return devicehub.Code_CODE_CONTINUE
	case dp.StatusCode_Terminate:
		return devicehub.Code_CODE_TERMINATE
	case dp.StatusCode_NotFount:
		return devicehub.Code_CODE_NOT_FOUNT
	case dp.StatusCode_BadRequest:
		return devicehub.Code_CODE_BAD_REQUEST
	case dp.StatusCode_MethodNotAllowed:
		return devicehub.Code_CODE_METHOD_NOT_ALLOWED
	case dp.StatusCode_TooManyRequests:
		return devicehub.Code_CODE_TOO_MANY_REQUESTS
	case dp.StatusCode_TooManyObservers:
		return devicehub.Code_CODE_TOO_MANY_OBSERVERS
	}
	log.Error().Str("devcie.status", code.String()).Msg("device status code not mapped")
	return devicehub.Code_CODE_INTERNAL_SERVER_ERROR
}

func (s *AccessServer) CoPost(ctx context.Context, req *devicehub.CoReq) (*devicehub.CoResp, error) {

	resp := &devicehub.CoResp{
		Id: req.Id,
	}
	session, ok := s.sessions.Get(req.DeviceId)
	if !ok {
		log.Warn().Uint32("reqid", req.Id).Err(devicetcp.ErrSessionNotFound).Msg("Post")
		resp.Code = devicehub.Code_CODE_DEVICEID_OFFLINE
		return resp, nil
	}
	if len(req.Data) > int(session.BodyCapSize-dp.HeaderLen_CoResp) {
		log.Error().Uint32("reqid", req.Id).Err(devicetcp.ErrOverCapacity).Msg("Post")
		resp.Code = devicehub.Code_CODE_BAD_REQUEST
		return resp, nil
	}
	uri := rtioutil.URIHash(req.Uri)
	code, data, err := session.Send(ctx, uri, dp.Method_ConstrainedPost, req.Data, 10*time.Second)
	if err != nil {
		if err == devicetcp.ErrSendTimeout {
			log.Error().Err(err).Msg("Post")
			resp.Code = devicehub.Code_CODE_REQUEST_TIMEOUT
			return resp, nil
		}
		log.Error().Err(err).Msg("Post")
		resp.Code = devicehub.Code_CODE_INTERNAL_SERVER_ERROR
		return resp, nil
	}
	resp.Code = transToRPCCode(code)
	if resp.Code == devicehub.Code_CODE_OK {
		resp.Data = data
	}
	return resp, nil
}

func (s *AccessServer) ObGet(req *devicehub.ObGetReq, stream devicehub.AccessService_ObGetServer) error {

	resp := &devicehub.ObGetResp{
		Id:  req.Id,
		Fid: 0,
	}
	session, ok := s.sessions.Get(req.DeviceId)
	if !ok {
		log.Warn().Uint32("reqid", req.Id).Err(devicetcp.ErrSessionNotFound).Msg("Obsevation init")
		resp.Code = devicehub.Code_CODE_DEVICEID_OFFLINE
		stream.Send(resp)
		return nil
	}
	if len(req.Data) > int(session.BodyCapSize-dp.HeaderLen_ObGetEstabReq) {
		log.Error().Uint32("reqid", req.Id).Err(devicetcp.ErrOverCapacity).Msg("Obsevation init")
		resp.Code = devicehub.Code_CODE_BAD_REQUEST
		stream.Send(resp)
		return nil
	}

	ob, err := session.CreateObserva()
	if err != nil {
		log.Error().Uint32("reqid", req.Id).Err(err).Msg("Obsevation create")
		return err
	}
	defer session.DestroyObserva(ob.ObserverID)

	log.Info().Uint32("reqid", req.Id).Uint16("obid", ob.ObserverID).Msg("Obsevation created")

	uri := rtioutil.URIHash(req.Uri)
	statusCode, err := session.ObGetEstablish(stream.Context(), uri, ob, req.Data, time.Second*20)
	if err != nil {
		log.Error().Uint32("reqid", req.GetId()).Err(err).Msg("Obsevation establish")
		if devicetcp.ErrSendTimeout == err {
			resp.Code = devicehub.Code_CODE_DEVICEID_TIMEOUT
		} else {
			resp.Code = devicehub.Code_CODE_INTERNAL_SERVER_ERROR
		}
		stream.Send(resp)
		return nil
	}

	if statusCode != dp.StatusCode_Continue {
		resp.Code = transToRPCCode(statusCode)
		log.Info().Uint32("reqid", req.GetId()).Err(err).Str("devcie.status", statusCode.String()).Msg("Obsevation establish result (exclude Continue):")
		stream.Send(resp)
		return nil
	}

	obGetNotifyServe(ob, req, stream)
	return nil
}
func obGetNotifyServe(ob *devicetcp.Observa, req *devicehub.ObGetReq, stream devicehub.AccessService_ObGetServer) {

	for {
		select {
		case <-ob.SessionDoneChan:
			log.Info().Uint32("reqid", req.Id).Msg("ObGet device session done")
			resp := &devicehub.ObGetResp{
				Id:   req.Id,
				Fid:  ob.FrameID,
				Code: devicehub.Code_CODE_DEVICEID_OFFLINE,
			}
			stream.Send(resp)
			return
		case <-stream.Context().Done():
			log.Info().Uint32("reqid", req.Id).Msg("ObGet stream context done")
			return
		case notifyReq, ok := <-ob.NotifyChan:
			resp := &devicehub.ObGetResp{
				Id:  req.Id,
				Fid: ob.FrameID,
			}
			if !ok {
				log.Info().Uint32("reqid", req.Id).Msg("ObGet ob.NotifyChan closed")
				resp.Code = devicehub.Code_CODE_TERMINATE
			} else {
				if notifyReq.Code == deviceproto.StatusCode_Continue &&
					notifyReq.Data != nil {
					resp.Data = notifyReq.Data
				}
				resp.Code = transToRPCCode(notifyReq.Code)
			}
			stream.Send(resp)
			ob.FrameID = ob.FrameID + 1
			if resp.Code != devicehub.Code_CODE_CONTINUE {
				return
			}
		}
	}
}

func InitRPCServer(ctx context.Context, addr string,
	sessionMap *devicetcp.SessionMap, wait *sync.WaitGroup) error {

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error().Err(err).Msg("listen failed")
		return err
	}
	s := grpc.NewServer()
	devicehub.RegisterAccessServiceServer(s, &AccessServer{sessions: sessionMap})

	go func() {
		<-ctx.Done()
		log.Debug().Msg("context done")
		s.GracefulStop()
	}()

	wait.Add(1)
	go func() {
		defer wait.Done()
		err = s.Serve(listener)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Warn().Msg("listener error closed")
			} else {
				log.Error().Err(err).Msg("serve failed")
			}
		}
		if listener != nil {
			listener.Close()
			log.Debug().Msg("listener closed")
		}
		log.Info().Msg("serve stoped")
	}()

	return nil
}

func (s *AccessServer) DeviceQuery(ctx context.Context, req *devicehub.DeviceQueryReq) (*devicehub.DeviceQueryResp, error) {

	resp := &devicehub.DeviceQueryResp{
		Id: req.Id,
	}
	session, ok := s.sessions.Get(req.DeviceId)
	if !ok {
		log.Warn().Uint32("reqid", req.Id).Err(devicetcp.ErrSessionNotFound).Msg("Get")
		resp.Code = devicehub.Code_CODE_NOT_FOUNT
		return resp, nil
	}
	resp.BodyCapSize = uint32(session.BodyCapSize)
	resp.RemoteAddr = session.RemoteAddr.String()
	resp.Code = devicehub.Code_CODE_OK
	return resp, nil
}
