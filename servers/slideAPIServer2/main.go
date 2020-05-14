package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/slides/v1"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func main() {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/presentations")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := slides.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Slides client: %v", err)
	}
	// slides.NewPresentationsPagesService(srv).GetThumbnail("asc","dcadv").
	// Prints the number of slides and elements in a sample presentation:
	// https://docs.google.com/presentation/d/1EAYk18WDjIG-zp_0vLm3CsfQh_i8eXc67Jo2O9C6Vuc/edit
	presentationId := "fileId"
	presentation, err := srv.Presentations.Get(presentationId).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from presentation: %v", err)
	}

	fmt.Printf("The presentation contains %d slides:\n", len(presentation.Slides))
	for i, slide := range presentation.Slides {
		fmt.Printf("- Slide #%d contains %d elements.\n", (i + 1),
			len(slide.PageElements))
		fmt.Println("page - ", i, " , objId - ", slide.ObjectId)
		thumb, err := srv.Presentations.Pages.GetThumbnail(presentationId, "objectId").ThumbnailPropertiesMimeType("PNG").Do()

		if err != nil {
			log.Fatalf("Unable to retrieve data from presentation: %v", err)
		}
		fmt.Println("url", thumb.ContentUrl)
		client := &http.Client{}
		req, err := http.NewRequest("GET", thumb.ContentUrl, nil)
		if err != nil {
			fmt.Println("Error in making get request", err)
		}
		res, err := client.Do(req)
		if err != nil {
			fmt.Println("Error while getting res", err)
		}
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println("Error while reading bytes", err)
		}
		var filename string
		filename = "poc" + strconv.Itoa(i+1) + ".png"
		fmt.Println("name", filename)
		err = ioutil.WriteFile(filename, body, 0644)
		if err != nil {
			fmt.Println("Error while creating File", err)
		}
	}
}
