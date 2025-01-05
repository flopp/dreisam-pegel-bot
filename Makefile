all:
	@echo "run -> run local cli"
	@echo "run-bot -> run local bot (this will create a mastodon post)"
	@echo "deploy -> build & deploy"

.PHONY: run
run:
	go run cmd/cli/main.go .data

.PHONY: run-bot
run-bot:
	go run cmd/bot/main.go production-config.json .data

.bin/bot-linux: cmd/bot/main.go internal/pegel/*.go go.mod 
	mkdir -p .bin
	GOOS=linux GOARCH=amd64 go build -o .bin/bot-linux cmd/bot/main.go

.PHONY: deploy
deploy: .bin/bot-linux
	ssh echeclus.uberspace.de mkdir -p packages/dreisampegelbot
	scp production-config.json scripts/cronjob.sh .bin/bot-linux echeclus.uberspace.de:packages/dreisampegelbot
	ssh echeclus.uberspace.de chmod +x packages/dreisampegelbot/cronjob.sh packages/dreisampegelbot/bot-linux
