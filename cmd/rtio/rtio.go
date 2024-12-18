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
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"text/template"

	"github.com/mkrainbow/rtio/internal/devicehub/server/apprpc"
	"github.com/mkrainbow/rtio/internal/devicehub/server/backendconn"
	"github.com/mkrainbow/rtio/internal/devicehub/server/configer"
	"github.com/mkrainbow/rtio/internal/devicehub/server/devicetcp"
	"github.com/mkrainbow/rtio/internal/httpaccess/server/httpgw"
	"github.com/mkrainbow/rtio/pkg/config"
	"github.com/mkrainbow/rtio/pkg/logsettings"

	"github.com/google/gops/agent"
	"github.com/rs/zerolog/log"
)

func main() {
	tcpAddr := flag.String("deviceaccess.addr", "0.0.0.0:17017", "address for device conntection")
	httpAddr := flag.String("httpaccess.addr", "0.0.0.0:17917", "address for http conntection")
	rpcAddr := flag.String("backend.rpc.addr", "0.0.0.0:17018", "(optional) address for app-server conntection")

	logFormat := flag.String("log.format", "text", "text or json")
	logLevel := flag.String("log.level", "warn", " debug, info, warn, error")
	deviceVerifier := flag.String("backend.deviceverifier", "http://localhost:17217/deviceverifier", "device verifier service address, for device verifier")
	hubConfiger := flag.String("backend.hubconfiger", "http://localhost:17317/hubconfiger", "service address for hub config")

	disableDeviceVerify := flag.Bool("disable.deviceverify", false, "disable the backend device verify config service")
	disableHubConfiger := flag.Bool("disable.hubconfiger", false, "disable the backend hub config service")

	enableHubTLS := flag.Bool("enable.hub.tls", false, "enable device hub TLS access")
	hubCertFile := flag.String("hub.tls.certfile", "", "TLS cert file for device hub")
	hubKeyFile := flag.String("hub.tls.keyfile", "", "TLS key file for device hub")

	enableHTTPS := flag.Bool("enable.https", false, "enable https gateway")
	httpsCertFile := flag.String("https.certfile", "", "TLS cert file")
	httpsKeyFile := flag.String("https.keyfile", "", "TLS key file")

	enableJWT := flag.Bool("enable.jwt", false, "enable the JWT validation")
	ed25519 := flag.String("jwt.ed25519", "", "the public key (pem) for JWT")

	completionBash := flag.Bool("completion-bash", false, "print bash autocomplete script")

	flag.Usage = printUsage
	flag.Parse()

	// handle completion-bash flag
	if *completionBash {
		printCompletionBash()
		os.Exit(0)
	}

	// set configs
	config.StringKV.Set("backend.deviceverifier", *deviceVerifier)
	config.StringKV.Set("backend.hubconfiger", *hubConfiger)
	config.BoolKV.Set("disable.deviceverify", *disableDeviceVerify)
	config.BoolKV.Set("disable.hubconfiger", *disableHubConfiger)

	// set log format and level
	logsettings.Set(*logFormat, *logLevel)

	// enable jwt
	if *enableJWT {
		if *ed25519 == "" {
			log.Error().Msg("JWT public key is empty")
			return
		}
		config.BoolKV.Set("enable.jwt", true)
		config.StringKV.Set("jwt.ed25519", *ed25519)
	}

	// show configs
	for _, v := range config.StringKV.List() {
		log.Debug().Msgf("Config:%s", v)
	}

	if err := agent.Listen(agent.Options{}); err != nil {
		log.Fatal().Err(err)
	}
	defer agent.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Info().Msg("rtio starting ...")

	backendconn.InitBackendConnn()

	wait := &sync.WaitGroup{}
	sessionMap := &devicetcp.SessionMap{}
	var err error
	if *enableHubTLS {
		err = devicetcp.InitTLSServer(ctx, *tcpAddr, sessionMap, wait, *hubCertFile, *hubKeyFile)
		if err != nil {
			log.Error().Err(err).Msg("Init TLS Server error")
			return
		}
	} else {
		err = devicetcp.InitTCPServer(ctx, *tcpAddr, sessionMap, wait)
		if err != nil {
			log.Error().Err(err).Msg("Init TCP Server error")
			return
		}
	}

	err = apprpc.InitRPCServer(ctx, *rpcAddr, sessionMap, wait)
	if err != nil {
		log.Error().Err(err).Msg("Init RPC Server error")
		return
	}

	if *enableHTTPS {
		err = httpgw.InitHttpsGateway(ctx, *rpcAddr, *httpAddr, wait, *httpsCertFile, *httpsKeyFile)
		if err != nil {
			log.Error().Err(err).Msg("Init Https Gateway error")
			return
		}
	} else {
		err = httpgw.InitHttpGateway(ctx, *rpcAddr, *httpAddr, wait)
		if err != nil {
			log.Error().Err(err).Msg("Init Http Gateway error")
			return
		}
	}

	if !*disableHubConfiger {
		configer.HubConfigerInit(ctx, wait)
	}

	log.Debug().Msg("rtio wait for subroutes")
	wait.Wait()
	log.Info().Msg("rtio stoped")
}
func printUsage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n\n", os.Args[0])
	fmt.Fprintln(flag.CommandLine.Output(), `  Generate a bash completion script with '-completion-bash'.Source it directly in your shell using:
    source <(`+os.Args[0]+` -completion-bash)`)
	fmt.Fprintln(flag.CommandLine.Output())
	flag.PrintDefaults()
}

type CompleteBash struct {
	Command string
	Flags   string
}

const (
	tmpBash string = `# Bash completion for {{.Command}}
_{{.Command}}_complete() {
    local cur short_opts long_opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Define base flags
    local flags=({{.Flags}})

    # Define flag-specific values
    declare -A flag_values
    flag_values["log.level"]="debug info warn error"
    flag_values["log.format"]="text json"

    # Generate single-dash and double-dash options
    short_opts=$(printf " -%s" "${flags[@]}")
    long_opts=$(printf " --%s" "${flags[@]}")

    # If completing after a flag, provide its specific values
    if [[ "${prev}" == "-log.level" || "${prev}" == "--log.level" ]]; then
        COMPREPLY=( $(compgen -W "${flag_values[log.level]}" -- "${cur}") )
        return 0
    fi
    if [[ "${prev}" == "-log.format" || "${prev}" == "--log.format" ]]; then
        COMPREPLY=( $(compgen -W "${flag_values[log.format]}" -- "${cur}") )
        return 0
    fi

    # Match options based on prefix
    if [[ "${cur}" == --* ]]; then
        COMPREPLY=( $(compgen -W "${long_opts}" -- "${cur}") )
        return 0
    elif [[ "${cur}" == -* ]]; then
        COMPREPLY=( $(compgen -W "${short_opts}" -- "${cur}") )
        return 0
    fi
}
complete -o default -F _{{.Command}}_complete {{.Command}}
`
)

func printCompletionBash() {
	var flagList string
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		flagList = flagList + "\"" + f.Name + "\" "
	})
	flagList = flagList + "\"h\" " + "\"help\""

	t, err := template.New("complete-bash").Parse(tmpBash)

	if err != nil {
		fmt.Println("parse template:", err)
		return
	}
	rtioBash := &CompleteBash{Command: "rtio", Flags: flagList}

	err = t.Execute(os.Stdout, rtioBash)
	if err != nil {
		fmt.Println("executing template:", err)
	}
}
