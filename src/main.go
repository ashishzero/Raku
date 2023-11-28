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
		log.Fatalln("error: token file not present")
	}
	botctx.RegisterApplicationCommand(SeasonalCommand)
	botctx.RegisterApplicationCommand(SearchAnimeCommand)
	botctx.RegisterApplicationCommand(SearchMangaCommand)
	botctx.Login(strings.TrimSpace(string(token)))
}
