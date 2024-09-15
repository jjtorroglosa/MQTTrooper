
.PHONY: mac linux all mqttrooper

mac: dist dist/mqttrooper.amd64.darwin
linux: dist dist/mqttrooper.amd64.linux

all: mac linux

mqttrooper: dist dist/mqttrooper.amd64.linux dist/mqttrooper.amd64.darwin

LINUX = GOARCH=amd64 GOOS=linux go
MAC = GOARCH=amd64 GOOS=darwin go
MAIN_FILES = main.go mqtt.go http.go yaml.go executor.go

dist/mqttrooper.amd64.darwin: $(MAIN_FILES)
	$(MAC) build -o $@ $^

dist/mqttrooper.amd64.linux: $(MAIN_FILES)
	$(LINUX) build -o $@ $^

dist:
	mkdir -p dist

clean:
	rm -rf dist

deploy: linux
	rsync -avz ./dist/ bell:services/mqttrooper/
	rsync -avz ./templates/ bell:services/mqttrooper/templates/
