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

	ds "github.com/mkrainbow/rtio/internal/devicehub/client/devicesession"
	"github.com/mkrainbow/rtio/pkg/logsettings"
)

func main() {

	logsettings.Set("text", "info")
	serverAddr := flag.String("server", "localhost:17017", "server address")
	deviceID := flag.String("id", "cfa09baa-4913-4ad7-a936-3e26f9671b09", "deviceid")
	deviceSecret := flag.String("secret", "mb6bgso4EChvyzA05thF9+wH", "devicesecret")
	flag.Parse()

	session, err := ds.Connect(context.Background(), *deviceID, *deviceSecret, *serverAddr)

	if err != nil {
		log.Println(err)
		return
	}

	session.RegisterPostHandler("/rainbow", func(req []byte) ([]byte, error) {
		log.Printf("%s", string(req))
		return []byte("world"), nil

	})

	session.Serve(context.Background())

	// do other things
	time.Sleep(time.Hour * 8760)

}
