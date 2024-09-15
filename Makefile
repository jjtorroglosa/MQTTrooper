
.PHONY: mac linux all mqtt-commander

mac: dist dist/mqtt-commander.amd64.darwin
linux: dist dist/mqtt-commander.amd64.linux

all: mac linux

mqtt-commander: dist dist/mqtt-commander.amd64.linux dist/mqtt-commander.amd64.darwin

LINUX = GOARCH=amd64 GOOS=linux go
MAC = GOARCH=amd64 GOOS=darwin go
MAIN_FILES = main.go mqtt.go http.go yaml.go executor.go

dist/mqtt-commander.amd64.darwin: $(MAIN_FILES)
	$(MAC) build -o $@ $^

dist/mqtt-commander.amd64.linux: $(MAIN_FILES)
	$(LINUX) build -o $@ $^

dist:
	mkdir -p dist

clean:
	rm -rf dist

deploy: linux
	rsync -avz ./dist/ bell:services/mqtt-commander/
	rsync -avz ./templates/ bell:services/mqtt-commander/templates/
