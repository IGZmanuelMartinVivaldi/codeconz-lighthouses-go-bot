SHELL := /bin/bash
SERVER_PORT  := 50051
BOT_PORT	 := 3001
BOT_NAME	 := go-bot1

runbotgo:
	go run ./src/bot.go -bn=$(BOT_NAME) -la=:$(BOT_PORT) -gs=:$(SERVER_PORT)

.PHONY: runbotgo
