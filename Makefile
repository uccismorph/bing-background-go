target_package := github.com/uccismorph/bing-background-go/cmd
config_file := bing/config.json

all:
	@mkdir -p bin
	@go build -o bin/bing_background $(target_package)
	@cp bing/config.json bin/config.json

.PHONY: clean
clean:
	@rm -rf ./bin