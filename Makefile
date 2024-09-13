
.PHONY: deploy

systemd-api: *.go
	mkdir -p build
	docker compose run --rm dev go build -o build/systemd-api *.go

deploy:
	rsync -avz ./ bell:services/systemd-api/
