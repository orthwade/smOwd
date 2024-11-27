package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Structure for the request body
type GraphQLRequest struct {
	Query string `json:"query"`
}

// Structure for the response body
type AnimeResponse struct {
	Data struct {
		Animes []struct {
			English  string `json:"english"`
			Russian  string `json:"russian"`
			Japanese string `json:"japanese"`
			ID       string `json:"id"`
			URL      string `json:"url"`
		} `json:"animes"`
	} `json:"data"`
}

func SearchAnime(name string) {
	// GraphQL query
	query := fmt.Sprintf(`
		query {
			animes(search: "%s") { 
				english
				russian
				japanese
				id
				url
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
		log.Fatal(err)
	}

	// Send the POST request
	url := "https://shikimori.one/api/graphql" // Make sure the URL is correct for the Shkimori API
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBodyJSON))
	if err != nil {
		log.Fatal(err)
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")

	// Send the request using the HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Parse the response into the AnimeResponse struct
	var animeResp AnimeResponse
	if err := json.Unmarshal(respBody, &animeResp); err != nil {
		log.Fatal(err)
	}

	// Print the response data
	for _, anime := range animeResp.Data.Animes {
		fmt.Printf("English: %s\n", anime.English)
		fmt.Printf("Russian: %s\n", anime.Russian)
		fmt.Printf("Japanese: %s\n", anime.Japanese)
		fmt.Printf("ID: %s\n", anime.ID)
		fmt.Printf("URL: %s\n", anime.URL)
	}
}
