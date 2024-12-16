all: debug

release:
	mkdir -p out
	go build -ldflags="-s -w" -o ./out/ github.com/mkrainbow/rtio/cmd/...

	mkdir -p out/examples
	if [ ! -d ./out/examples/certificates ]; then cp -v -r ./examples/certificates ./out/examples/; fi	
	go build -ldflags="-s -w" -o ./out/examples/ github.com/mkrainbow/rtio/examples/...
	
debug:
	mkdir -p out
	go build -gcflags="all=-N -l" -o ./out/ github.com/mkrainbow/rtio/cmd/...

	mkdir -p out/examples
	if [ ! -d ./out/examples/certificates ]; then cp -v -r ./examples/certificates ./out/examples/; fi	
	go build -gcflags="all=-N -l" -o ./out/examples/ github.com/mkrainbow/rtio/examples/...

clean:
	go clean -i github.com/mkrainbow/rtio/...
	rm -rf ./out 
	

.PHONY: \
	all \
	release \
	debug \
	clean 
