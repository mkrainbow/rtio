ORG_ID  := 906b73c
ORG_VER := v0.8.0
BUILD_TIME := $(shell date --iso-8601=seconds)

GOFLAGS_RELEASE := -ldflags "-s -w -X 'main.Time=$(BUILD_TIME)' -X 'main.ID=$(ORG_ID)' -X 'main.Ver=$(ORG_VER)'"
GOFLAGS_DEBUG   := -gcflags="all=-N -l" -ldflags "-X 'main.Time=$(BUILD_TIME)' -X 'main.ID=$(ORG_ID)' -X 'main.Ver=$(ORG_VER)'"

all: debug

release:
	mkdir -p out
	go build $(GOFLAGS_RELEASE) -o ./out/ github.com/mkrainbow/rtio/cmd/...

	@ mkdir -p out/examples
	@ if [ ! -d ./out/examples/certificates ]; then cp -v -r ./examples/certificates ./out/examples/; fi	
	go build $(GOFLAGS_RELEASE) -o ./out/examples/ github.com/mkrainbow/rtio/examples/...

	@ # for internal test
	@ mkdir -p out/devicehub
	@ go build $(GOFLAGS_RELEASE) -o ./out/devicehub github.com/mkrainbow/rtio/internal/devicehub/...
	

debug:
	mkdir -p out
	go build $(GOFLAGS_DEBUG) -o ./out/ github.com/mkrainbow/rtio/cmd/...

	@ mkdir -p out/examples
	@ if [ ! -d ./out/examples/certificates ]; then cp -v -r ./examples/certificates ./out/examples/; fi	
	go build $(GOFLAGS_DEBUG) -o ./out/examples/ github.com/mkrainbow/rtio/examples/...

	@ # for internal test
	@ mkdir -p out/devicehub
	@ go build $(GOFLAGS_DEBUG) -o ./out/devicehub github.com/mkrainbow/rtio/internal/devicehub/...

test:
	go test -count=1  -timeout 7m github.com/mkrainbow/rtio/...

clean:
	go clean -i github.com/mkrainbow/rtio/...
	rm -rf ./out 

.PHONY: \
	all \
	release \
	debug \
	test \
	clean 

