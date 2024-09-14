
.PHONY: deploy http mqtt mac

mac: dist dist/http.amd64.darwin dist/mqtt.amd64.darwin
linux: dist dist/http.amd64.linux dist/mqtt.amd64.linux

all: mac linux

http: dist dist/http.amd64.linux dist/http.amd64.darwin
mqtt: dist dist/mqtt.amd64.linux dist/mqtt.amd64.darwin

#LINUX = docker compose run --rm dev go
LINUX = GOARCH=amd64 GOOS=linux go
MAC = GOARCH=amd64 GOOS=darwin go
HTTP_FILES = main.go http.go yaml.go cmd.go
MQTT_FILES = mqtt.go cmd.go yaml.go

dist/mqtt.amd64.darwin: $(MQTT_FILES)
	$(MAC) build -o $@ $^

dist/mqtt.amd64.linux:$(MQTT_FILES)
	$(LINUX) build -o $@ $^

dist/http.amd64.darwin: $(HTTP_FILES)
	$(MAC) build -o $@ $^

dist/http.amd64.linux: $(HTTP_FILES)
	$(LINUX) build -o $@ $^

dist:
	mkdir -p dist

clean:
	rm -rf dist

deploy:
	rsync -avz ./ bell:services/systemd-api/
