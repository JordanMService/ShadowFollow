package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	// import third party libraries
	"github.com/PuerkitoBio/goquery"
	"github.com/fabioberger/airtable-go"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type InstaShadow struct {
	Id     string
	Fields struct {
		UserName      string
		PostCount     string
		Subscriptions string
	}
}

type Config struct {
	SENDGRID_API_KEY string
	AIRTABLE_API_KEY string
	BASE_ID          string
	FROM_EMAIL       string
}

func main() {

	config := Config{}
	file, _ := os.Open("config.json")
	decoder := json.NewDecoder(file)
	decoder.Decode(&config)

	airtableAPIKey := config.AIRTABLE_API_KEY
	baseID := config.BASE_ID

	client, _ := airtable.New(airtableAPIKey, baseID)

	shadows := []InstaShadow{}

	client.ListRecords("PostCount", &shadows)

	for _, shadow := range shadows {
		instaUrl := fmt.Sprintf("https://www.instagram.com/%s/?hl=en", shadow.Fields.UserName)
		doc, _ := goquery.NewDocument(instaUrl)
		content := doc.Find("meta")
		postCount := ""
		content.EachWithBreak(func(index int, item *goquery.Selection) bool {
			metaContent, _ := item.Attr("content")
			if strings.Contains(metaContent, "Posts") {
				regex, _ := regexp.Compile(`(\d+)+(?: Posts)`)
				match := regex.FindStringSubmatch(metaContent)
				if len(match) > 1 {
					postCount = match[1]
					return false
				}
			}
			return true
		})

		if postCount != "" && postCount != shadow.Fields.PostCount {
			i, _ := strconv.Atoi(shadow.Fields.PostCount)
			UpdatedFields := map[string]interface{}{
				"PostCount": strconv.Itoa(i + 1),
			}
			if err := client.UpdateRecord("PostCount", shadow.Id, UpdatedFields, &shadow); err != nil {
				panic(err)
			}
			subs := strings.Split(shadow.Fields.Subscriptions, ",")
			for _, sub := range subs {
				SendEmail(shadow, sub, config.FROM_EMAIL, config.SENDGRID_API_KEY)
			}
		}
	}
}

func SendEmail(shadow InstaShadow, email string, fromEmail string, sendGridKey string) {
	from := mail.NewEmail("Stalker Robot", fromEmail)
	subject := fmt.Sprintf("%s has made a new instagram post", shadow.Fields.UserName)
	to := mail.NewEmail("Subsciber", email)
	plainTextContent := fmt.Sprintf("https://www.instagram.com/%s/?hl=en", shadow.Fields.UserName)
	htmlContent := fmt.Sprintf("https://www.instagram.com/%s/?hl=en", shadow.Fields.UserName)
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(sendGridKey)
	response, err := client.Send(message)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
}
