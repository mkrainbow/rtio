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
	"log"
	"time"

	"github.com/mkrainbow/rtio-device-sdk-go/rtio"
)

func main() {

	serverAddr := flag.String("server", "localhost:17017", "server address")
	deviceID := flag.String("id", "cfa09baa-4913-4ad7-a936-3e26f9671b09", "deviceid")
	deviceSecret := flag.String("secret", "mb6bgso4EChvyzA05thF9+wH", "devicesecret")
	flag.Parse()

	session, err := rtio.Connect(context.Background(), *deviceID, *deviceSecret, *serverAddr)
	if err != nil {
		log.Println(err)
		return
	}

	session.Serve(context.Background())

	t := time.NewTicker(time.Second * 5)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			resp, err := session.CoPost(context.Background(), "/aa/bb", []byte("test for device post"), time.Second*20)
			if err != nil {
				log.Println(err)
			} else {
				log.Printf("resp=%s\n", string(resp))
			}
		}
	}

}
