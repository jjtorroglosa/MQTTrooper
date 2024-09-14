
.PHONY: deploy http mqtt mac

mac: dist/http.mac dist/mqtt.mac

http: dist/http.linux dist/http.mac
mqtt: dist/mqtt.mac dist/mqtt.linux

dist/mqtt.mac: *.go
	mkdir -p dist
	go build -o dist/mqtt.mac mqtt.go cmd.go yaml.go

dist/mqtt.linux: *.go
	mkdir -p dist
	docker compose run --rm dev go build -o dist/mqtt.linux mqtt.go cmd.go yaml.go

dist/http.mac: *.go
	mkdir -p dist
	go build -o dist/http.mac main.go http.go yaml.go cmd.go

dist/http.linux: *.go
	mkdir -p dist
	docker compose run --rm dev go build -o dist/http.linux main.go http.go yaml.go cmd.go

systemd-api: *.go
	docker compose run --rm dev go build -o build/systemd-api *.go

deploy:
	rsync -avz ./ bell:services/systemd-api/
