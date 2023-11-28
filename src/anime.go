package main

import (
	"Raku/anilist"
	"Raku/botctx"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var SearchAnimeCommand = botctx.CommandDesc{
	Name:        "anime-search",
	Description: "Search for anime",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "search",
			Description: "tags to search for anime",
			Required:    true,
		},
	},
	Func:        animeSearch,
	Interaction: mediaSearchInteract,
}

var SearchMangaCommand = botctx.CommandDesc{
	Name:        "manga-search",
	Description: "Search for manga",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "search",
			Description: "tags to search for manga",
			Required:    true,
		},
	},
	Func:        mangaSearch,
	Interaction: mediaSearchInteract,
}

var SeasonalCommand = botctx.CommandDesc{
	Name:        "anime-seasonal",
	Description: "List of seasonal anime",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "year",
			Description: "Released year",
			Required:    false,
			MaxValue:    5000,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "season",
			Description: "Season",
			Required:    false,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{
					Name:  "all",
					Value: "ALL",
				},
				{
					Name:  "winter",
					Value: "WINTER",
				},
				{
					Name:  "spring",
					Value: "SPRING",
				},
				{
					Name:  "summer",
					Value: "SUMMER",
				},
				{
					Name:  "fall",
					Value: "FALL",
				},
			},
		},
	},
	Func:        animeSeasonal,
	Interaction: animeSeasonalInteract,
}

func animeSearch(session *discordgo.Session, i *discordgo.InteractionCreate) {
	doMediaSearch(session, i, "anime")
}

func mangaSearch(session *discordgo.Session, i *discordgo.InteractionCreate) {
	doMediaSearch(session, i, "manga")
}

func mediaSearchInteract(session *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
	if len(args) != 1 {
		return
	}

	id, _ := strconv.ParseInt(args[0], 10, 32)

	embed := &discordgo.MessageEmbed{
		Title:       "N/A",
		Description: "N/A",
		Type:        discordgo.EmbedTypeRich,
		Color:       0xdd1111,
	}
	media, err := anilist.FindMedia(session.Client, int(id))

	if err == nil {
		embed = createMediaEmbed(media)
	}

	body := &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	}

	res := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: body,
	}

	err = session.InteractionRespond(i.Interaction, res)
	if err != nil {
		log.Printf("error: %+v\n", err)
	}
}

func animeSeasonal(session *discordgo.Session, i *discordgo.InteractionCreate) {
	now := time.Now()
	page := 1
	year := now.Year()
	season := anilist.MonthToSeason(now.Month())

	data := i.ApplicationCommandData()
	for _, option := range data.Options {
		if option.Name == "year" {
			year = int(option.IntValue())
		} else if option.Name == "season" {
			season = anilist.StringToSeason(option.StringValue())
		}
	}

	body := doSeasonalAnime(session, page, 0, year, season)

	res := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: body,
	}

	err := session.InteractionRespond(i.Interaction, res)
	if err != nil {
		log.Printf("error: %+v\n", err)
	}
}

func animeSeasonalInteract(session *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
	data := i.MessageComponentData()

	if len(args) != 3 {
		return
	}

	page, _ := strconv.ParseInt(args[0], 10, 32)
	year, _ := strconv.ParseInt(args[1], 10, 32)
	season, _ := strconv.ParseInt(args[2], 10, 32)
	var index int64

	if data.Values != nil {
		index, _ = strconv.ParseInt(data.Values[0], 10, 32)
	}

	body := doSeasonalAnime(session, int(page), int(index), int(year), anilist.Season(season))

	res := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: body,
	}

	err := session.InteractionRespond(i.Interaction, res)
	if err != nil {
		log.Printf("error: %+v\n", err)
	}
}

//
//
//

func createMediaEmbed(media *anilist.Media) *discordgo.MessageEmbed {
	var color int64
	if len(media.CoverImage.Color) > 1 {
		hex := media.CoverImage.Color[1 : len(media.CoverImage.Color)-1]
		color, _ = strconv.ParseInt(hex, 16, 32)
	}

	description := "N/A"

	if len(media.Description) > 1 {
		description = media.Description
		description = strings.ReplaceAll(description, "<i>", "*")
		description = strings.ReplaceAll(description, "</i>", "*")
		description = strings.ReplaceAll(description, "<b>", "**")
		description = strings.ReplaceAll(description, "</b>", "**")
		description = strings.ReplaceAll(description, "<u>", "__")
		description = strings.ReplaceAll(description, "</u>", "__")
		description = strings.ReplaceAll(description, "<br>", "")
		if len(description) > 512 {
			description = description[0:510] + ".."
		}
	}

	var builder strings.Builder

	for idx, val := range media.Genres {
		builder.WriteString(val)
		if idx < len(media.Genres)-1 {
			builder.WriteString(",")
		}
	}

	alt := "N/A"

	if len(media.Title.English) > 1 {
		alt = media.Title.English
	}

	start := media.StartDate
	end := media.EndDate

	date := "N/A"

	if start.Year != 0 {
		date = fmt.Sprintf("%02d/%02d/%04d", start.Day, start.Month, start.Year)

		if end.Year != 0 {
			date += fmt.Sprintf(" to %02d/%02d/%04d", end.Day, end.Month, end.Year)
		}
	}

	var releaseName = ""
	var releaseCount = 0

	if media.Type == "ANIME" {
		releaseName = "Episodes"
		releaseCount = media.Episodes
	} else {
		releaseName = "Volumes"
		releaseCount = media.Volumes
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Alt",
			Value:  alt,
			Inline: true,
		},
		{
			Name:   "Aired",
			Value:  date,
			Inline: true,
		},
		{
			Name:   "\u0000",
			Value:  "\u0000",
			Inline: true,
		},
		{
			Name:   "Score",
			Value:  fmt.Sprint(media.MeanScore),
			Inline: true,
		},
		{
			Name:   "Status",
			Value:  media.Status,
			Inline: true,
		},
		{
			Name:   releaseName,
			Value:  fmt.Sprint(releaseCount),
			Inline: true,
		},
		{
			Name:  "Genres",
			Value: builder.String(),
		},
	}

	embed := discordgo.MessageEmbed{
		Title:       media.Title.Romaji,
		URL:         media.SiteUrl,
		Description: description,
		Type:        discordgo.EmbedTypeRich,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: media.CoverImage.Large},
		Color:       int(color),
		Fields:      fields,
	}
	return &embed
}

func encodeSeasonalAnimeId(page float64, year int, season anilist.Season) string {
	return fmt.Sprintf("anime-seasonal;%v;%v;%v", int(page), year, int(season))
}

func doMediaSearch(session *discordgo.Session, i *discordgo.InteractionCreate, mediaType string) {
	search := "example"

	data := i.ApplicationCommandData()
	for _, option := range data.Options {
		if option.Name == "search" {
			search = option.StringValue()
		}
	}

	page, err := anilist.SearchMedia(session.Client, mediaType, search, 3)

	embed := &discordgo.MessageEmbed{
		Title:       "N/A",
		Description: "N/A",
		Type:        discordgo.EmbedTypeRich,
		Color:       0xdd1111,
	}

	buttons := make([]discordgo.MessageComponent, 0, 3)

	if err == nil {
		fields := make([]*discordgo.MessageEmbedField, len(page.Media))

		for idx, media := range page.Media {
			buttons = append(buttons, discordgo.Button{
				Label:    fmt.Sprint(idx + 1),
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("%v-search;%v", mediaType, media.Id),
			})
			fields[idx] = &discordgo.MessageEmbedField{
				Name:  fmt.Sprintf("%v. %v", idx+1, media.Title.Romaji),
				Value: media.Title.English,
			}
		}
		buttons = append(buttons, discordgo.Button{
			Label: "Open",
			Style: discordgo.LinkButton,
			URL:   page.URL,
		})

		embed = &discordgo.MessageEmbed{
			Title:  "Results for " + search,
			Type:   discordgo.EmbedTypeRich,
			Color:  0x11dddd,
			Fields: fields,
		}
	}

	body := &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	}

	if len(buttons) > 0 {
		body.Components = []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: buttons,
			},
		}
	}

	res := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: body,
	}

	err = session.InteractionRespond(i.Interaction, res)
	if err != nil {
		log.Printf("error: %+v\n", err)
	}
}

func doSeasonalAnime(session *discordgo.Session, currentPage int, index int, year int, season anilist.Season) *discordgo.InteractionResponseData {
	info := anilist.PageInfo{
		CurrentPage: float64(currentPage),
		PerPage:     16,
	}

	page, err := anilist.FindSeasonal(session.Client, info, season, year)

	if err != nil {
		log.Printf("err: %+v\n", err)
		return nil
	}

	if len(page.Media) == 0 {
		data := &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Error",
					Description: "No seasonal anime found",
					Color:       0xdd1111,
				},
			},
		}
		return data
	}

	if index >= len(page.Media) {
		index = 0
	}

	var options []discordgo.SelectMenuOption

	for idx, media := range page.Media {
		label := media.Title.Romaji

		if len(label) > 50 {
			label = label[0:50]
		}

		option := discordgo.SelectMenuOption{
			Label:       label,
			Value:       fmt.Sprint(idx),
			Description: fmt.Sprintf("%02d/%02d", media.StartDate.Month, media.StartDate.Year),
		}
		options = append(options, option)
	}

	buttons := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.StringSelectMenu,
					CustomID:    encodeSeasonalAnimeId(page.PageInfo.CurrentPage, year, season),
					Placeholder: page.Media[index].Title.Romaji,
					Options:     options,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    fmt.Sprintf("Page %v", page.PageInfo.CurrentPage),
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("anime-seasonal-page-%v", page.PageInfo.CurrentPage),
					Disabled: true,
				},
				discordgo.Button{
					Label:    "Prev",
					Style:    discordgo.PrimaryButton,
					CustomID: encodeSeasonalAnimeId(page.PageInfo.CurrentPage-1, year, season),
					Disabled: info.CurrentPage == 1,
				},
				discordgo.Button{
					Label:    "Next",
					Style:    discordgo.PrimaryButton,
					CustomID: encodeSeasonalAnimeId(info.CurrentPage+1, year, season),
					Disabled: !page.PageInfo.HasNextPage,
				},
				discordgo.Button{
					Label: "Open",
					Style: discordgo.LinkButton,
					URL:   page.URL,
				},
			},
		},
	}

	embed := &discordgo.MessageEmbed{
		Title:       "N/A",
		Description: "N/A",
		Type:        discordgo.EmbedTypeRich,
		Color:       0xdd1111,
	}
	media, err := anilist.FindMedia(session.Client, page.Media[index].Id)

	if err == nil {
		embed = createMediaEmbed(media)
	}

	data := &discordgo.InteractionResponseData{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: buttons,
	}

	return data
}
