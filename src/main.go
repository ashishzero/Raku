package main

import (
	"Raku/botctx"
	"log"
	"os"
	"strings"
)

func main() {
	token, err := os.ReadFile("token")
	if err != nil {
		log.Fatalln("error: token file not present. please add a text file named token with discord bot token in it")
	}
	botctx.RegisterApplicationCommand(SeasonalCommand)
	botctx.RegisterApplicationCommand(SearchAnimeCommand)
	botctx.RegisterApplicationCommand(SearchMangaCommand)
	botctx.RegisterApplicationCommand(MigrateCommand)
	botctx.Login(strings.TrimSpace(string(token)))
}
