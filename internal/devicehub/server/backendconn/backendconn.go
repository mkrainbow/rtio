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

package backendconn

import (
	"errors"

	"github.com/mkrainbow/rtio/internal/devicehub/server/service"
	"github.com/mkrainbow/rtio/internal/devicehub/server/verifier"
	"github.com/mkrainbow/rtio/pkg/config"

	"github.com/rs/zerolog/log"
)

var (
	verifyClient    *verifier.Client
	ErrVerifyClient = errors.New("Failed to get device verify client")

	serviceClient    *service.Client
	ErrServiceClient = errors.New("Failed to get device service client")
)

func InitBackendConnn() {

	disableVerify := config.BoolKV.GetWithDefault("disable.deviceverify", false)
	if !disableVerify {
		url, ok := config.StringKV.Get("backend.deviceverifier")
		if !ok {
			log.Error().Msg("device service URL empty")
		}
		verifyClient = verifier.NewClient(url)
	}

	serviceClient = service.NewClient()
}

func GetDeviceVerifier() (*verifier.Client, error) {

	if verifyClient != nil {
		return verifyClient, nil
	}
	return nil, ErrVerifyClient
}
func GetServiceClient() (*service.Client, error) {

	if serviceClient != nil {
		return serviceClient, nil
	}
	return nil, ErrServiceClient
}
