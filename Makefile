
.PHONY: deploy

systemd-api: *.go
	docker compose run --rm dev go build -o systemd-api *.go

deploy:
	rsync -avz ./ bell:services/systemd-api/