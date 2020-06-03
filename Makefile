target_package := github.com/uccmorph/bing-background-go/cmd
config_file := bing/config.json

all:
	@mkdir -p bin
	@go build -o bin/bing_background $(target_package)
	@cp bing/config.json bin/config.json
	@cp record/record.db.json bin/record.db

.PHONY: clean
clean:
	@rm -rf ./bin

.PHONY: install
install:
	@cp -f bin/bing_background ~/go/bin/