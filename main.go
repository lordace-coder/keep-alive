package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/robfig/cron/v3"
)

type Link struct {
	ID    string `json:"id"`
	USER  string `json:"user"`
	URL   string `json:"url"`
	TITLE string `json:"title"`
	ERROR string `json:"error"`
}

type ResponseWrapper struct {
	Items []Link `json:"items"`
}

func main() {
	app := fiber.New()
	app.Use(logger.New())

	app.Static("/", "./templates/dist")
	// Schedule a task to run every 5 minutes
	c := cron.New()
	c.AddFunc("*/7 * * * *", func() {
		var response ResponseWrapper

		// get a list of all links from db
		resp, err := http.Get("https://keep-alive.pockethost.io/api/collections/links/records")
		if err != nil {
			log.Println("Error fetching links:", err)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(body, &response); err != nil {
			panic(err)
		}
		// loop through all links and check if they are alive
		for _, link := range response.Items {
			resp, err := http.Get(link.URL)
			if err != nil {
				log.Println("Error fetching link:", err)
				continue
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				log.Println("Link is down:", link.URL)
				link.ERROR = "Link is down: " + link.URL
				jsonData, err := json.Marshal(link)
				if err != nil {
					log.Println("Error marshalling link:", err)
				}
				// send a ERROR to the user
				http.Post("https://keep-alive.pockethost.io/api/collections/links/records/"+link.ID, "application/json", bytes.NewBuffer(jsonData))
			} else {
				log.Println("Link is up:", link.URL)
			}
		}

	})
	c.Start()

	app.Get("*", func(c *fiber.Ctx) error {
		return c.SendFile("./dist/index.html")
	})
	port := os.Getenv("PORT")

	app.Listen(":" + port)
}
