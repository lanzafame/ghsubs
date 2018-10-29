package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/google/go-github/github"
	"github.com/tidwall/gjson"
	"github.com/tomnomnom/linkheader"
	"golang.org/x/oauth2"
)

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("please set GITHUB_TOKEN env var with a Personal Access Token")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	user, _, err := client.Users.Get(ctx, "lanzafame")
	if err != nil {
		log.Fatal(err)
	}
	subsURL := *user.SubscriptionsURL

	rch := make(chan string, 10)

	go func() {
		for {
			select {
			case repo := <-rch:
				if !gjson.Valid(repo) {
					log.Fatal("json response not valid")
				}
				result := gjson.Parse(repo)
				result.ForEach(func(key, value gjson.Result) bool {
					name := value.Get("full_name")
					fmt.Println(name)
					return true // keep iterating
				})
			}
		}
	}()

	for {
		subsURL, err = appendQueryParams(subsURL)
		if err != nil {
			log.Fatal(err)
		}
		resp, err := http.Get(subsURL)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Fatal(resp)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		rch <- string(body)

		links := linkheader.Parse(resp.Header.Get("link"))
		nexts := links.FilterByRel("next")
		var next linkheader.Link
		if len(nexts) > 0 {
			next = nexts[0]
		}
		lasts := links.FilterByRel("last")
		var last linkheader.Link
		if len(lasts) > 0 {
			last = lasts[0]
		}
		if next.URL == last.URL {
			break
		}
		subsURL = next.URL
	}

}

func appendQueryParams(u string) (string, error) {
	// as we aren't going via the github client anymore
	token := os.Getenv("GITHUB_TOKEN")
	subsURL, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	q := subsURL.Query()
	q.Set("per_page", "100")
	q.Set("access_token", token)
	subsURL.RawQuery = q.Encode()
	return subsURL.String(), nil
}
