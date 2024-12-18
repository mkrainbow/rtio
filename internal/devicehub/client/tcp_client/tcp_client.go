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
	"context"
	"flag"
	"os/signal"

	ds "github.com/mkrainbow/rtio/internal/devicehub/client/devicesession"
	"github.com/mkrainbow/rtio/pkg/logsettings"

	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

func copostHandler(req []byte) ([]byte, error) {
	log.Info().Str("req", string(req)).Msg("")
	return []byte("world!"), nil
}
func copost128Handler(req []byte) ([]byte, error) {
	log.Info().Str("req", string(req)).Msg("uri len=128,")
	return []byte("world!"), nil
}

func obgetHandler(ctx context.Context, req []byte) (<-chan []byte, error) {
	log.Info().Str("req", string(req)).Msg("")
	respChan := make(chan []byte, 1)
	go func(context.Context, <-chan []byte) {

		defer func() {
			close(respChan)
			log.Info().Msg("Observer exit")
		}()
		t := time.NewTicker(time.Millisecond * 300)
		defer t.Stop()
		i := 0
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("ctx.Done()")
				return
			case <-t.C:
				log.Info().Msg("Notify")
				respChan <- []byte("world! " + strconv.Itoa(i))
				i++
				if i >= 10 {
					return
				}
			}
		}
	}(ctx, respChan)

	return respChan, nil
}

func virtalDeviceRun(ctx context.Context, wait *sync.WaitGroup, deviceID, deviceSecret, serverAddr string) {
	defer wait.Done()

	log.Info().Str("deviceid", deviceID).Msg("virtalDeviceRun run")

	defer func() {
		log.Info().Str("deviceid", deviceID).Msg("virtalDeviceRun exit")
	}()

	session, err := ds.Connect(ctx, deviceID, deviceSecret, serverAddr)
	if err != nil {
		log.Error().Str("deviceid", deviceID).Err(err).Msg("connection error")
		return
	}
	session.RegisterObGetHandler("/test", obgetHandler)
	session.RegisterPostHandler("/test", copostHandler)
	session.RegisterPostHandler("/0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456", copost128Handler)

	session.Serve(ctx)

	t := time.NewTicker(time.Second * 5)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("ctx done")
			return
		case <-t.C:
			log.Debug().Str("virtalDeviceRun now", time.Now().String())
			resp, err := session.CoPost(ctx, "/aa/bb", []byte("test for device post"), time.Second*20)

			if err != nil {
				log.Error().Err(err).Msg("")
			} else {
				log.Info().Str("resp", string(resp)).Msg("")
			}
		}
	}

}

func main() {
	logsettings.Set("text", "debug")
	serverAddr := flag.String("server", "localhost:17017", "server address")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	wait := &sync.WaitGroup{}
	wait.Add(1)
	go virtalDeviceRun(ctx, wait, "cfa09baa-4913-4ad7-a936-2e26f9671b05", "mb6bgso4EChvyzA05thF9+wH", *serverAddr)

	wait.Wait()
	log.Error().Msg("client exit")

}
