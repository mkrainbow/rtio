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

	"strconv"
	"time"

	ds "github.com/mkrainbow/rtio/internal/devicehub/client/devicesession"
	"github.com/mkrainbow/rtio/pkg/logsettings"

	"github.com/rs/zerolog/log"
)

func handler(ctx context.Context, req []byte) (<-chan []byte, error) {
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

func main() {

	logsettings.Set("text", "info")
	serverAddr := flag.String("server", "localhost:17017", "server address")
	deviceID := flag.String("id", "cfa09baa-4913-4ad7-a936-3e26f9671b09", "deviceid")
	deviceSecret := flag.String("secret", "mb6bgso4EChvyzA05thF9+wH", "devicesecret")
	flag.Parse()

	session, err := ds.Connect(context.Background(), *deviceID, *deviceSecret, *serverAddr)

	if err != nil {
		log.Error().Err(err).Msg("")
	}

	session.RegisterObGetHandler("/rainbow", handler)
	session.Serve(context.Background())

	// do other things
	time.Sleep(time.Hour * 8760)

}
