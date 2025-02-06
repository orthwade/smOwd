package animes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"smOwd/logs"
	"time"
)

const url = "https://shikimori.one/api/graphql"

type GraphQLRequest struct {
	Query string `json:"query"`
}

var client = &http.Client{Timeout: 10 * time.Second}

type Anime struct {
	ShikiID       string `json:"id"`
	MalID         string `json:"malId"`
	English       string `json:"english"`
	Japanese      string `json:"japanese"`
	Status        string `json:"status"`
	Episodes      int    `json:"episodes"`
	EpisodesAired int    `json:"episodesAired"`
	URL           string `json:"url"`
}

type AnimeResponse struct {
	Data struct {
		Animes []Anime `json:"animes"`
	} `json:"data"`
}

func SearchAnimeByName(ctx context.Context, name string) ([]Anime, error) {
	logger := logs.DefaultFromCtx(ctx)

	query := fmt.Sprintf(` query{
		animes(search: "%s", limit: 50) {
			id       
			malId    
			english   
			japanese 
			status
			episodes 
			episodesAired
			url
		}
	}`, name)

	reqBody := GraphQLRequest{Query: query}

	reqBodyJson, err := json.Marshal(reqBody)

	if err != nil {
		logger.Error("Failed to marshall query", "error", err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url,
		bytes.NewBuffer(reqBodyJson))

	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		logger.Fatal("Failed to create requet", "error", err)
	}

	resp, err := client.Do(req)

	if err != nil {
		logger.Fatal("Failed request", "error", err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		logger.Fatal("Failed to read response", "error", err)
	}

	var animeResponse AnimeResponse

	err = json.Unmarshal(respBody, &animeResponse)

	if err != nil {
		logger.Fatal("Failed to unmarshall", "error", err)
	}

	return animeResponse.Data.Animes, nil
}
