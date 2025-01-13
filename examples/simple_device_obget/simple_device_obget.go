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
	"context"
	"flag"
	"log"

	"strconv"
	"time"

	"github.com/mkrainbow/rtio-device-sdk-go/rtio"
)

func handler(ctx context.Context, req []byte) (<-chan []byte, error) {

	log.Printf("received [%s] and reply [world! *]", string(req))

	respChan := make(chan []byte, 1)
	go func(context.Context, <-chan []byte) {

		defer func() {
			close(respChan)
			log.Println("Observer exit")
		}()
		t := time.NewTicker(time.Millisecond * 300)
		defer t.Stop()
		i := 0
		for {
			select {
			case <-ctx.Done():
				log.Println("ctx.Done()")
				return
			case <-t.C:
				log.Println("Notify")
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

	serverAddr := flag.String("server", "localhost:17017", "server address")
	deviceID := flag.String("id", "cfa09baa-4913-4ad7-a936-3e26f9671b09", "deviceid")
	deviceSecret := flag.String("secret", "mb6bgso4EChvyzA05thF9+wH", "devicesecret")
	flag.Parse()

	session, err := rtio.Connect(context.Background(), *deviceID, *deviceSecret, *serverAddr)

	if err != nil {
		log.Printf("Connect error, err=%s", err.Error())
	}

	session.RegisterObGetHandler("/rainbow", handler)
	session.Serve(context.Background())

	// do other things
	time.Sleep(time.Hour * 8760)

}
