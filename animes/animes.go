package animes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"smOwd/logs"
	"unicode"

	"strings"
	"time"
)

const url = "https://shikimori.one/api/graphql"

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

var client = &http.Client{Timeout: 10 * time.Second}

type Anime struct {
	ShikiID       string `json:"id"`
	MalID         string `json:"malId"`
	English       string `json:"english"`
	Russian       string `json:"russian"`
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
	Errors []struct {
		Message string `json: "message"`
	} `json:"errors"`
}

func splitWords(input string) []string {
	// Replace underscores with spaces and then split by spaces
	input = strings.ReplaceAll(input, "_", " ")
	words := strings.Fields(input)
	return words
}

func isWholeWord(word, sentence string) bool {
	sentenceRunes := []rune(sentence)
	wordRunes := []rune(word)
	wordLen := len(wordRunes)

	for i := 0; i <= len(sentenceRunes)-wordLen; i++ {
		// Check if the substring matches the word
		if string(sentenceRunes[i:i+wordLen]) == word {
			// Ensure it's a whole word by checking surrounding characters
			isStartBoundary := i == 0 || !unicode.IsLetter(sentenceRunes[i-1])
			isEndBoundary := i+wordLen == len(sentenceRunes) || !unicode.IsLetter(sentenceRunes[i+wordLen])
			if isStartBoundary && isEndBoundary {
				return true
			}
		}
	}
	return false
}

// containsAllWords checks if all words in the input exist as whole words in the sentence
func containsAllWords(input, sentence string) bool {
	words := splitWords(input)

	// Normalize case for case-insensitive matching
	sentence = strings.ToLower(sentence)

	for _, word := range words {
		if !isWholeWord(strings.ToLower(word), sentence) {
			return false
		}
	}
	return true
}

func SearchAnimeByName(ctx context.Context, name string) ([]Anime, error) {
	logger := logs.DefaultFromCtx(ctx)

	query := fmt.Sprintf(` query{
		animes(search: "%s", limit: 500) {
			id       
			malId    
			english 
			russian  
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

	if len(animeResponse.Errors) > 0 {
		for _, Error := range animeResponse.Errors {
			logger.Error("GraphQL error", "message", Error.Message)
		}
	}

	var result []Anime

	for _, a := range animeResponse.Data.Animes {
		if containsAllWords(name, a.English) {
			result = append(result, a)
		} else if containsAllWords(name, a.Russian) {
			result = append(result, a)
		}
	}

	return result, nil
}

func SearchAnimeByShikiIDs(ctx context.Context, shikiIDs []string) ([]Anime, error) {
	logger := logs.DefaultFromCtx(ctx)

	idsString := strings.Join(shikiIDs, ",")

	query := fmt.Sprintf(` query($ids: String!) {
		animes(ids: $ids, limit: %d) {
			id       
			malId    
			english
			russian   
			japanese 
			status
			episodes 
			episodesAired
			url
		}
	}`, len(shikiIDs))

	reqBody := GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"ids": idsString,
		}}

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

	if len(animeResponse.Errors) > 0 {
		for _, Error := range animeResponse.Errors {
			logger.Error("GraphQL error", "message", Error.Message)
		}
	}

	return animeResponse.Data.Animes, nil
}
