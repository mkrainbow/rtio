all: debug

release:
	mkdir -p out
	go get github.com/mkrainbow/rtio-device-sdk-go

	go build -ldflags="-s -w" -o ./out/ github.com/mkrainbow/rtio/cmd/...

	@ mkdir -p out/examples
	@ if [ ! -d ./out/examples/certificates ]; then cp -v -r ./examples/certificates ./out/examples/; fi	
	go build -ldflags="-s -w" -o ./out/examples/ github.com/mkrainbow/rtio/examples/...

	@ # for internal test
	@ mkdir -p out/devicehub
	@ go build -ldflags="-s -w" -o ./out/devicehub github.com/mkrainbow/rtio/internal/devicehub/...
	

debug:
	mkdir -p out
	go get github.com/mkrainbow/rtio-device-sdk-go
	
	go build -gcflags="all=-N -l" -o ./out/ github.com/mkrainbow/rtio/cmd/...

	@ mkdir -p out/examples
	@ if [ ! -d ./out/examples/certificates ]; then cp -v -r ./examples/certificates ./out/examples/; fi	
	go build -gcflags="all=-N -l" -o ./out/examples/ github.com/mkrainbow/rtio/examples/...

	@ # for internal test
	@ mkdir -p out/devicehub
	@ go build -gcflags="all=-N -l" -o ./out/devicehub github.com/mkrainbow/rtio/internal/devicehub/...

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
