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
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

type ServerTLS struct {
	listener net.Listener
	// cert       *tls.Certificate
	config     *tls.Config
	sessions   *SessionMap
	wait       *sync.WaitGroup
	sessionNum int32
}

func NewServerTLS(addr string, sessionMap *SessionMap, certFile, keyFile string) (*ServerTLS, error) {

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Error().Err(err).Msg("LoadX509KeyPair failed")
		return nil, err
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	listener, err := tls.Listen("tcp", addr, config)
	if err != nil {
		log.Error().Err(err).Msg("listen failed")
		return nil, err
	}
	return &ServerTLS{
		listener:   listener,
		config:     config,
		sessions:   sessionMap,
		wait:       &sync.WaitGroup{},
		sessionNum: 0,
	}, nil
}

func (s *ServerTLS) AddSession(ctx context.Context, deviceID string, session *Session) {
	invalid, ok := s.sessions.Get(deviceID)
	if ok {
		invalid.Cancel()
		log.Debug().Msg("cancel old session")
		<-invalid.Done()
		log.Debug().Msg("old session done")
	}
	atomic.AddInt32(&s.sessionNum, 1)
	s.sessions.Set(deviceID, session)
}
func (s *ServerTLS) DelSession(deviceID string) {
	s.sessions.Del(deviceID)
	atomic.AddInt32(&s.sessionNum, -1)
}

func (s *ServerTLS) Shutdown() {
	log.Info().Msg("shutdown")
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *ServerTLS) Serve(c context.Context) {
	ctx, cancel := context.WithCancel(c)
	log.Info().Str("addr", s.listener.Addr().String()).Msg("server started")

	go func() {
		t := time.NewTicker(time.Second * 10)
	EXIT_LOOPY:
		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("context done")
				break EXIT_LOOPY
			case <-t.C:
				// log.Error().Int32("sessionnum", s.sessionNum).Msg("") // for stress
			}
		}
		s.Shutdown()
	}()

	s.wait.Add(1)
	go func() {
		defer s.wait.Done()
		defer cancel()

		for s.listener != nil {
			conn, err := s.listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					log.Warn().Msg("listener error closed")
				} else {
					log.Error().Err(err).Msg("listener accept error")
				}
				break
			}
			s.wait.Add(1)
			session := newSession(conn)
			go session.serve(ctx, s.wait, s.AddSession, s.DelSession)
		}
		log.Info().Msg("listener closed")
	}()

	log.Info().Msg("waiting for subroutes")
	if s.wait != nil {
		s.wait.Wait()
	}
	log.Info().Msg("waiting end")
}

func InitTLSServer(ctx context.Context,
	addr string,
	sessionMap *SessionMap,
	wait *sync.WaitGroup,
	certFile, keyFile string) error {

	log.Info().Msg("TLS access enabled")
	log.Debug().Str("certfile", certFile).Str("keyfile", keyFile).Msg("TLS certfile and keyfile")
	s, err := NewServerTLS(addr, sessionMap, certFile, keyFile)
	if err != nil {
		log.Error().Err(err).Msg("NewServerTLS error")
		return err
	}
	wait.Add(1)
	go func() {
		defer wait.Done()
		s.Serve(ctx)
	}()
	return nil
}
