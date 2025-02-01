package animes

import (
	"bytes"
	"context"

	"encoding/json"
	"fmt"
	"io/ioutil"

	// "log/slog"
	"net/http"
	// "os"
	"smOwd/logs"
	"strconv"
	"time"
)

const url = "https://shikimori.one/api/graphql"

type GraphQLRequest struct {
	Query string `json:"query"`
}

var client = &http.Client{Timeout: 10 * time.Second}

type AnimeResponse struct {
	Data struct {
		Animes []struct {
			ShikiID       string `json:"id"`
			MalId         string `json:"malId"`
			English       string `json:"english"`
			Japanese      string `json:"japanese"`
			Status        string `json:"status"`
			Episodes      int    `json:"episodes"`
			EpisodesAired int    `json:"episodesAired"`
		} `json:"animes"`
	} `json:"data"`
}

func SearchAnimeByName(ctx context.Context, name string) ([]Anime, error) {
	var sliceAnime []Anime

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
	defer resp.Body.Close()

	if err != nil {
		logger.Fatal("Failed request", "error", err)
	}

	respBody, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		logger.Fatal("Failed to read response", "error", err)
	}

	var animeResponse AnimeResponse

	err = json.Unmarshal(respBody, &animeResponse)

	if err != nil {
		logger.Fatal("Failed to unmarshall", "error", err)
	}
	for _, anime := range animeResponse.Data.Animes {

		shikiIdInt, err := strconv.Atoi(anime.ShikiID)

		if err != nil {
			logger.Error("Failed to convert shiki ID to integer", "error", err)
			return nil, err
		}

		malIdInt, err := strconv.Atoi(anime.MalId)

		if err != nil {
			logger.Error("Failed to convert Mal ID to integer", "error", err)
			return nil, err
		}

		sliceAnime = append(sliceAnime, Anime{
			ID:            -1,
			ShikiID:       shikiIdInt,
			MalID:         malIdInt,
			English:       anime.English,
			Japanese:      anime.Japanese,
			Status:        anime.Status,
			Episodes:      anime.Episodes,
			EpisodesAired: anime.EpisodesAired})
	}

	return sliceAnime, nil
}
