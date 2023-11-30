package main

import (
	"Raku/botctx"
	"log"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]

	_, fp := os.Stat("token")

	if len(args) == 0 && fp != nil {
		log.Fatalln("error: please provide bot token. you can pass token as command line argument or make a file named token with the bot token in it")
	}

	token := ""

	if len(args) > 0 {
		token = args[0]
	} else {
		content, _ := os.ReadFile("token")
		token = string(content)
	}

	botctx.RegisterApplicationCommand(SeasonalCommand)
	botctx.RegisterApplicationCommand(SearchAnimeCommand)
	botctx.RegisterApplicationCommand(SearchMangaCommand)
	botctx.RegisterApplicationCommand(MigrateCommand)
	botctx.Login(strings.TrimSpace(token))
}
