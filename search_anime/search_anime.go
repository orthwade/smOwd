package search_anime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"
	"strconv"
)

// Structure for the request body
type GraphQLRequest struct {
	Query string `json:"query"`
}

// Structure for the response body
type AnimeResponse struct {
	Data struct {
		Animes []struct {
			English       string `json:"english"`
			Russian       string `json:"russian"`
			Japanese      string `json:"japanese"`
			ID            string `json:"id"`
			URL           string `json:"url"`
			Status        string `json:"status"`
			Episodes      int    `json:"episodes"`
			EpisodesAired int    `json:"episodesAired"`
		} `json:"animes"`
	} `json:"data"`
}

func SearchAnimeById(ctx context.Context, ID int64) (AnimeResponse, error) {
	// Retrieve the logger from the context
	logger, ok := ctx.Value("logger").(*slog.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	strID := strconv.FormatInt(ID, 10)

	// GraphQL query
	query := fmt.Sprintf(`
		query {
			animes(ids: "%s") { 
				english
				russian
				japanese
				id
				url
				status
				episodes
				episodesAired
			}
		}
	`, strID)

	// Prepare the GraphQL request body
	reqBody := GraphQLRequest{
		Query: query,
	}

	// Marshal the request body into JSON
	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		logger.Error("Failed to marshal request body", "error", err)
		return AnimeResponse{}, err
	}

	// Send the POST request
	url := "https://shikimori.one/api/graphql"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBodyJSON))
	if err != nil {
		logger.Error("Failed to create HTTP request", "error", err)
		return AnimeResponse{}, err
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("HTTP request failed", "error", err)
		return AnimeResponse{}, err
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body", "error", err)
		return AnimeResponse{}, err
	}

	// Parse the response into the AnimeResponse struct
	var animeResp AnimeResponse
	if err := json.Unmarshal(respBody, &animeResp); err != nil {
		logger.Error("Failed to unmarshal response", "error", err)
		return AnimeResponse{}, err
	}

	logger.Info("Anime retrieved", "ID", strID, "Animes", animeResp.Data.Animes)

	return animeResp, nil
}

func SearchAnimeByName(ctx context.Context, name string) (AnimeResponse, error) {
	// Retrieve the logger from the context
	logger, ok := ctx.Value("logger").(*slog.Logger)
	if !ok {
		// If the logger is not found in the context, fall back to a default logger
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	// GraphQL query
	query := fmt.Sprintf(`
		query {
			animes(search: "%s", limit: 10) { 
				english
				russian
				japanese
				id
				url
				status
				episodes
				episodesAired
			}
		}
	`, name)

	// Prepare the GraphQL request body
	reqBody := GraphQLRequest{
		Query: query,
	}

	// Marshal the request body into JSON
	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		logger.Error("Failed to marshal request body", "error", err)
		return AnimeResponse{}, err
	}

	// Send the POST request
	url := "https://shikimori.one/api/graphql" // Make sure the URL is correct for the Shikimori API
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBodyJSON))
	if err != nil {
		logger.Error("Failed to create HTTP request", "error", err)
		return AnimeResponse{}, err
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("HTTP request failed", "error", err)
		return AnimeResponse{}, err
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body", "error", err)
		return AnimeResponse{}, err
	}

	// Parse the response into the AnimeResponse struct
	var animeResp AnimeResponse
	if err := json.Unmarshal(respBody, &animeResp); err != nil {
		logger.Error("Failed to unmarshal response", "error", err)
		return AnimeResponse{}, err
	}

	// Log and return the response data
	for _, anime := range animeResp.Data.Animes {
		logger.Info("Anime retrieved", "English", anime.English, "Russian", anime.Russian, "Japanese", anime.Japanese, "ID", anime.ID, "URL", anime.URL)
	}

	return animeResp, nil
}
