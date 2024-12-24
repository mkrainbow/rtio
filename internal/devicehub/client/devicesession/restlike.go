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

package devicesession

import (
	"context"
	"errors"
	"time"

	dp "github.com/mkrainbow/rtio/pkg/deviceproto"
	"github.com/mkrainbow/rtio/pkg/rtioutil"

	"github.com/rs/zerolog/log"
)

// Error Code for the registration interface.
var (
	ErrAlreadyRegistered = errors.New("ErrAlreadyRegistered")
)

// RegisterCoPostHandler registers a handler for CoPOST requests to the specified URI.Not Thread-safe.
func (s *DeviceSession) RegisterCoPostHandler(uri string, handler func(req []byte) ([]byte, error)) error {
	d := rtioutil.URIHash(uri)
	_, ok := s.regPostHandlerMap[d]
	if ok {
		return ErrAlreadyRegistered
	}
	s.regPostHandlerMap[d] = handler
	return nil
}

// RegisterObGetHandler registers a handler for ObGET requests to the specified URI.Not Thread-safe.
func (s *DeviceSession) RegisterObGetHandler(uri string, handler func(ctx context.Context, req []byte) (<-chan []byte, error)) error {
	d := rtioutil.URIHash(uri)
	_, ok := s.regObGetHandlerMap[d]
	if ok {
		return ErrAlreadyRegistered
	}

	s.regObGetHandlerMap[d] = handler
	return nil
}

// Error Code for the REST-Like layer.
var (
	ErrInternel            = errors.New("ErrInternel")
	ErrInternalServerError = errors.New("ErrInternalServerError")
	ErrResourceNotFount    = errors.New("ErrResourceNotFount")
	ErrBadRequest          = errors.New("ErrBadRequest")
	ErrMethodNotAllowed    = errors.New("ErrMethodNotAllowed")
	ErrTooManyRequests     = errors.New("ErrTooManyRequests")
	ErrRequestTimeout      = errors.New("ErrRequestTimeout")
)

func transToSDKError(code dp.StatusCode) error {
	switch code {
	case dp.StatusCode_Unknown:
		return ErrInternalServerError
	case dp.StatusCode_InternalServerError:
		return ErrInternalServerError
	case dp.StatusCode_OK:
		return nil
	case dp.StatusCode_Continue:
		return ErrInternalServerError
	case dp.StatusCode_Terminate:
		return ErrInternalServerError
	case dp.StatusCode_NotFount:
		return ErrResourceNotFount
	case dp.StatusCode_BadRequest:
		return ErrBadRequest
	case dp.StatusCode_MethodNotAllowed:
		return ErrMethodNotAllowed
	case dp.StatusCode_TooManyRequests:
		return ErrTooManyRequests
	case dp.StatusCode_TooManyObservers:
		return ErrInternalServerError
	default:
		return ErrInternel
	}
}

// CoPost Sends a CoPost request to the specified URI with the given payload and timeout.
func (s *DeviceSession) CoPost(ctx context.Context, uri string, Req []byte, timeout time.Duration) ([]byte, error) {
	headerID := s.genHeaderID()
	d := rtioutil.URIHash(uri)
	respChan, err := s.sendCoReq(headerID, dp.Method_ConstrainedPost, d, Req)
	if err != nil {
		log.Error().Err(err).Msg("Post")
		return nil, ErrInternel
	}
	statusCode, data, err := s.receiveCoRespWithContext(ctx, headerID, respChan)
	if err != nil {
		if err == ErrSendTimeout {
			log.Error().Err(err).Msg("Post")
			return nil, ErrRequestTimeout
		}
		log.Error().Err(err).Msg("Post")
		return nil, ErrInternel
	}
	return data, transToSDKError(statusCode)
}
