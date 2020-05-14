package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

type slide struct {
	Url      string `json:"url"`
	Filename string `json:"fileName"`
	AuthCode string `json:"authCode"`
}

var config *oauth2.Config

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config, authCode string) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	// tokFile := "token.json"
	// tok, err := tokenFromFile(tokFile)
	// if err != nil {
	// tok := getTokenFromWeb(config,authCode)
	// saveToken(tokFile, tok)
	// }
	tok := extractToken(config, authCode)

	return config.Client(context.Background(), tok)
}

func getURL(config *oauth2.Config) string {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	return authURL
}

func extractToken(config *oauth2.Config, authCode string) *oauth2.Token {
	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
// func tokenFromFile(file string) (*oauth2.Token, error) {
// 	f, err := os.Open(file)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer f.Close()
// 	tok := &oauth2.Token{}
// 	err = json.NewDecoder(f).Decode(tok)
// 	return tok, err
// }

// Saves a token to a file path.
// func saveToken(path string, token *oauth2.Token) {
// 	fmt.Printf("Saving credential file to: %s\n", path)
// 	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
// 	if err != nil {
// 		log.Fatalf("Unable to cache oauth token: %v", err)
// 	}
// 	defer f.Close()
// 	json.NewEncoder(f).Encode(token)
// }
func donwload(id string, filename string, authCode string) error {
	fmt.Println("config", config)
	client := getClient(config, authCode)
	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
		return err
	}
	// srv.Files.Export
	// drive.NewFilesService(srv)

	res, err := srv.Files.Export(id, "application/pdf").Download()
	if err != nil {
		log.Fatalf("Unable to respond: %v", err)
		return err
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Unable to read content: %v", err)
		return err
	}
	file, err := os.OpenFile(
		filename,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		log.Fatal(err)
		return err
	}

	command := "pdf2svg " + filename + " test_%d.svg all"
	cmd := exec.Command("cmd", "/C", command)
	err = cmd.Run()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println("dir", cmd.Dir)
	if err != nil {
		log.Println("error while executing cli", err)
		return err
	}
	return nil
}
func main() {
	router := gin.Default()
	md := cors.DefaultConfig()
	md.AllowAllOrigins = true
	md.AllowHeaders = []string{"*"}
	md.AllowMethods = []string{"*"}
	md.ExposeHeaders = []string{"Authorization"}
	router.Use(cors.New(md))
	router.GET("/getAuthURL", getAuthURL())
	router.POST("/slideTosvg", slideTosvg())
	router.GET("/streamPPT", streamPPT())
	s := &http.Server{
		Addr:    ":4700",
		Handler: router,
	}
	s.ListenAndServe()
}

func getAuthURL() gin.HandlerFunc {
	return func(c *gin.Context) {
		b, err := ioutil.ReadFile("credentials.json")
		if err != nil {
			log.Fatalf("Unable to read client secret file: %v", err)
			return
		}

		// If modifying these scopes, delete your previously saved token.json.
		config, err = google.ConfigFromJSON(b, "https://www.googleapis.com/auth/drive")
		if err != nil {
			log.Fatalf("Unable to parse client secret file to config: %v", err)
			return
		}

		url := getURL(config)
		c.JSON(http.StatusOK, url)
		// c.Redirect(http.StatusTemporaryRedirect, url)
		// client = getClient(config)
		// c.JSON(http.StatusOK, "Converted successfully")
		return
	}
}

func slideTosvg() gin.HandlerFunc {
	return func(c *gin.Context) {
		slideData := slide{}
		fmt.Println("consifg", config)
		err := c.Bind(&slideData)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("url", slideData.Url)
		r, _ := regexp.Compile("/presentation/d/([a-zA-Z0-9-_]+)")
		x := r.FindString(slideData.Url)
		fmt.Println(x)
		r2 := regexp.MustCompile("/presentation/d/")
		fileId := r2.ReplaceAllString(x, "")
		fmt.Println(fileId)
		err = donwload(fileId, slideData.Filename+".pdf", slideData.AuthCode)
		if err != nil {
			log.Fatal(err)
			c.JSON(http.StatusOK, "Not able to convert to svg")
		}
		c.JSON(http.StatusOK, "Converted successfully")
		return
	}
}

func streamPPT() gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := ioutil.ReadFile("test.pdf")
		if err != nil {
			log.Fatal("err", err)
		}
		c.Data(200, "application/pdf", data)
	}
}
