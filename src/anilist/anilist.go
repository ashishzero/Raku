package anilist

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Season uint8

const (
	Winter Season = 1
	Spring Season = 2
	Summer Season = 3
	Fall   Season = 4
	All    Season = 5
)

var (
	seasonNames = map[Season]string{
		Winter: "WINTER",
		Spring: "SPRING",
		Summer: "SUMMER",
		Fall:   "FALL",
		All:    "ALL",
	}

	baseURL = "https://graphql.anilist.co/"
)

type Title struct {
	English string
	Romaji  string
}

type CoverImage struct {
	Large  string
	Medium string
	Color  string
}

type FuzzyDate struct {
	Year  int
	Month int
	Day   int
}

type Media struct {
	Title       Title
	Id          int
	Type        string
	CoverImage  CoverImage
	Description string
	SiteUrl     string
	Status      string
	StartDate   FuzzyDate
	EndDate     FuzzyDate
	Episodes    int
	Volumes     int
	Genres      []string
	MeanScore   int
}

type MediaMinimal struct {
	Title     Title
	Id        int
	StartDate FuzzyDate
}

type PageInfo struct {
	CurrentPage float64
	PerPage     float64
	HasNextPage bool
}

type Page struct {
	PageInfo PageInfo
	Media    []MediaMinimal
	URL      string
}

type postParam struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type errorLocation struct {
	Line   int
	Column int
}

type postError struct {
	Message   string
	Status    int
	Locations []errorLocation
}

type postDataFindSeasonal struct {
	Page `json:"Page"`
}

type postDataFindAnime struct {
	Media `json:"Media"`
}

type postReply[T any] struct {
	Errors []postError
	Data   *T
}

func buildError(errs []postError) error {
	var builder strings.Builder
	for _, err := range errs {
		builder.WriteString("Locations: [")
		for i, loc := range err.Locations {
			builder.WriteString(fmt.Sprintf("{ Ln %v, Col %v }", loc.Line, loc.Column))
			if i != len(err.Locations)-1 {
				builder.WriteString(",")
			}
			builder.WriteString(" ")
		}
		builder.WriteString("] Status: ")
		builder.WriteString(fmt.Sprint(err.Status))
		builder.WriteString(", Message: ")
		builder.WriteString(err.Message)
		builder.WriteString("\n")
	}
	return errors.New(builder.String())
}

func executeQuery[T any](client *http.Client, param postParam) (*T, error) {
	body, err := json.Marshal(param)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var reply postReply[T]

	err = json.NewDecoder(res.Body).Decode(&reply)
	if err != nil {
		return nil, err
	}

	if reply.Errors != nil {
		return nil, buildError(reply.Errors)
	}

	return reply.Data, nil
}

func StringToSeason(str string) Season {
	lstr := strings.ToUpper(str)
	for idx, name := range seasonNames {
		if lstr == name {
			return Season(idx)
		}
	}
	return All
}

func SeasonToString(season Season) string {
	return seasonNames[season]
}

func MonthToSeason(month time.Month) Season {
	season := All
	if month == time.December || month == time.January || month == time.February {
		season = Winter
	} else if month == time.March || month == time.April || month == time.May {
		season = Spring
	} else if month == time.June || month == time.July || month == time.August {
		season = Summer
	} else if month == time.September || month == time.October || month == time.November {
		season = Fall
	}
	return season
}

func FindMedia(client *http.Client, id int) (*Media, error) {
	q := `
	query ($id: Int) {
		Media (id: $id) {
			id,
			type,
			title {
				romaji,
				english
			}
			coverImage {
				large
				medium
				color
			}
			description (asHtml: false)
			siteUrl
			status
			startDate {
				year,
				month,
				day
			}
			endDate {
				year,
				month,
				day
			}
			episodes,
			volumes,
			genres,
			meanScore
		}
	}
	`

	param := postParam{
		Query: q,
		Variables: map[string]interface{}{
			"id": id,
		},
	}

	res, err := executeQuery[postDataFindAnime](client, param)
	if err != nil {
		return nil, err
	}

	return &res.Media, nil
}

func SearchMedia(client *http.Client, media string, search string, limit int) (*Page, error) {
	q := `
	query ($type: MediaType, $tags: String, $limit: Int) {
		Page (page: 1, perPage: $limit) {
			pageInfo {
				currentPage,
				perPage,
				hasNextPage
			}
		
			media (search: $tags, type: $type) {
				id,
				title {
					romaji,
					english
				}
				startDate {
					year,
					month,
					day
				}
			}
	  	}
	}
	`
	param := postParam{
		Query: q,
		Variables: map[string]interface{}{
			"type":  strings.ToUpper(media),
			"tags":  search,
			"limit": limit,
		},
	}

	res, err := executeQuery[postDataFindSeasonal](client, param)
	if err != nil {
		return nil, err
	}

	res.Page.URL = fmt.Sprintf("https://anilist.co/search/%v?search=%v&sort=SEARCH_MATCH", media, url.QueryEscape(search))

	return &res.Page, nil
}

func FindSeasonal(client *http.Client, page PageInfo, season Season, year int) (*Page, error) {
	q := `
	query ($page: Int, $perPage: Int, $season: MediaSeason, $year: Int) {
		Page (page: $page, perPage: $perPage) {
			pageInfo {
				currentPage,
				perPage,
				hasNextPage
			}
		
			media (season: $season, seasonYear: $year, type: ANIME, sort: POPULARITY_DESC) {
				id,
				title {
					romaji,
					english
				}
				startDate {
					year,
					month,
					day
				}
			}
	  	}
	}
	`

	param := postParam{
		Query: q,
		Variables: map[string]interface{}{
			"page":    page.CurrentPage,
			"perPage": page.PerPage,
			"year":    year,
		},
	}

	if season != All {
		param.Variables["season"] = seasonNames[season]
	}

	res, err := executeQuery[postDataFindSeasonal](client, param)
	if err != nil {
		return nil, err
	}

	if season != All {
		res.Page.URL = fmt.Sprintf("https://anilist.co/search/anime?year=%v&season=%v", year, seasonNames[season])
	} else {
		res.Page.URL = fmt.Sprintf("https://anilist.co/search/anime?year=%v", year)
	}

	return &res.Page, nil
}
