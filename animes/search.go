package animes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"
	"smOwd/logs"
)

// GraphQLRequest defines the structure for the GraphQL request body.
type GraphQLRequest struct {
	Query string `json:"query"`
}

// GraphQLResponse represents the response structure from Shikimori's API.
type GraphQLResponse struct {
	Data struct {
		Animes []Anime `json:"animes"`
	} `json:"data"`
}

// SearchAnimeByID queries the Shikimori API by anime ID.
func SearchAnimeByID(ctx context.Context, shikiID int) (*Anime, error) {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}
	// Build GraphQL query
	query := fmt.Sprintf(`
		query {
			animes(ids: "%d") { 
				id
				malId
				english
				japanese
				episodes
				episodesAired
			}
		}
	`, shikiID)

	// Execute the query
	response, err := executeQuery(ctx, query, logger)
	if err != nil {
		return nil, err
	}

	// Handle the response
	if len(response.Data.Animes) == 0 {
		logger.Warn("No anime found", "ShikiID", shikiID)
		return nil, nil // No anime found
	}

	anime := response.Data.Animes[0] // Assume the first result is the most relevant
	logger.Info("Anime found", "ShikiID", anime.ShikiID, "English", anime.English)
	return &anime, nil
}

// SearchAnimeByName queries the Shikimori API by anime name.
func SearchAnimeByName(ctx context.Context, name string) ([]Anime, error) {
	logger, ok := ctx.Value("logger").(*logs.Logger)
	if !ok {
		logger = logs.New(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	// Build GraphQL query
	query := fmt.Sprintf(`
		query {
			animes(search: "%s", limit: 10) { 
				id
				malId
				english
				japanese
				episodes
				episodesAired
			}
		}
	`, name)

	// Execute the query
	response, err := executeQuery(ctx, query, logger)
	if err != nil {
		return nil, err
	}

	// Handle the response
	if len(response.Data.Animes) == 0 {
		logger.Warn("No anime found", "Name", name)
		return nil, nil // No anime found
	}

	logger.Info("Animes retrieved", "Count", len(response.Data.Animes))
	return response.Data.Animes, nil
}

// Helper: Executes a GraphQL query and parses the response.
func executeQuery(ctx context.Context, query string, logger *logs.Logger) (*GraphQLResponse, error) {
	// Prepare request body
	reqBody := GraphQLRequest{
		Query: query,
	}
	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		logger.Error("Failed to marshal request body", "error", err)
		return nil, err
	}

	// Send HTTP request
	url := "https://shikimori.one/api/graphql"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBodyJSON))
	if err != nil {
		logger.Error("Failed to create HTTP request", "error", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{} // Reusable HTTP client instance
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("HTTP request failed", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Read and parse response
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body", "error", err)
		return nil, err
	}

	var gqlResp GraphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		logger.Error("Failed to unmarshal response", "error", err)
		return nil, err
	}

	return &gqlResp, nil
}
