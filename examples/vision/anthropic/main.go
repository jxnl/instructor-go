package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"github.com/liushuangls/go-anthropic/v2"
)

type Movie struct {
	Title string `json:"title"          jsonschema:"title=title,description=The title of the movie,required=true,example=Ex Machina"`
	Year  int    `json:"year,omitempty" jsonschema:"title=year,description=The year of the movie,required=false,example=2014"`
}

type MovieCatalog struct {
	Catalog []Movie `json:"catalog"`
}

func (bc *MovieCatalog) PrintCatalog() {
	fmt.Printf("Number of movies in the catalog: %d\n\n", len(bc.Catalog))
	for _, movie := range bc.Catalog {
		fmt.Printf("Title:  %s\n", movie.Title)
		if movie.Year != 0 {
			fmt.Printf("Year:   %d\n", movie.Year)
		}
		fmt.Println("--------------------")
	}
}

func main() {
	ctx := context.Background()

	client := instructor.FromAnthropic(
		anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY")),
		instructor.WithMode(instructor.ModeJSONSchema),
		instructor.WithMaxRetries(3),
	)

	url := "https://raw.githubusercontent.com/instructor-ai/instructor-go/main/examples/vision/anthropic/movies.jpg"
	data, err := urlToBase64(url)
	if err != nil {
		panic(err)
	}

	var movieCatalog MovieCatalog
	_, err = client.CreateMessages(ctx, anthropic.MessagesRequest{
		Model: "claude-3-haiku-20240307",
		Messages: []anthropic.Message{
			{
				Role: anthropic.RoleUser,
				Content: []anthropic.MessageContent{
					anthropic.NewImageMessageContent(anthropic.NewMessageContentSource(
						anthropic.MessagesContentSourceTypeBase64,
						"image/jpeg",
						data,
					)),
					anthropic.NewTextMessageContent("Extract the movie catalog from the screenshot"),
				},
			},
		},
		MaxTokens: 1000,
	},
		&movieCatalog,
	)
	if err != nil {
		panic(err)
	}

	movieCatalog.PrintCatalog()
	/*
		Number of movies in the catalog: 18

		Title:  Oppenheimer
		Year:   2023
		--------------------
		Title:  The Dark Knight
		Year:   2008
		--------------------
		Title:  Interstellar
		Year:   2014
		--------------------
		Title:  Inception
		Year:   2010
		--------------------
		Title:  Tenet
		Year:   2020
		--------------------
		Title:  Dunkirk
		Year:   2017
		--------------------
		Title:  Memento
		Year:   2000
		--------------------
		Title:  The Dark Knight Rises
		Year:   2012
		--------------------
		Title:  Batman Begins
		Year:   2005
		--------------------
		Title:  The Prestige
		Year:   2006
		--------------------
		Title:  Insomnia
		Year:   2002
		--------------------
		Title:  Following
		Year:   1998
		--------------------
		Title:  Man of Steel
		Year:   2013
		--------------------
		Title:  Transcendence
		Year:   2014
		--------------------
		Title:  Justice League
		Year:   2017
		--------------------
		Title:  Batman v Superman: Dawn of Justice
		Year:   2016
		--------------------
		Title:  Ending the Knight
		Year:   2016
		--------------------
		Title:  Larceny
		--------------------
	*/
}

/*
 * Image utilties
 */

func urlToBase64(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}
