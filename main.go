package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.mills.io/prologic/go-gopher"
	"github.com/gorilla/feeds"
)

func main() {
	listenAndServe("0.0.0.0:9000", "cosmic.voyage:70")
}

const MAX_RSS_FEED_ITEMS = 10

func newFeedItem(title string, url string) (item *feeds.Item, err error) {
	item = new(feeds.Item)
	item.Title = title
	item.Link = &feeds.Link{}
	item.Created = time.Now()

	body, err := renderPage(url)
	if err != nil {
		return
	}

	body = strings.Replace(body, "\n", "<br>", -1)
	item.Content = body
	return
}

func renderPage(url string) (body string, err error) {
	res, err := gopher.Get(
		fmt.Sprintf(
			"gopher://%s",
			url,
		),
	)
	if err != nil {
		return
	}

	if res.Body == nil {
		err = errors.New("no response body")
		return
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}

	body = string(bytes)
	return
}

func renderRss(w http.ResponseWriter, hostport string, d gopher.Directory) error {
	now := time.Now()
	feed := &feeds.Feed{
		Title:   hostport,
		Created: now,
		Link:    &feeds.Link{},
	}

	rss_item_count := 0

	for _, item := range d.Items {
		if item.Type == gopher.INFO && item.Selector == "TITLE" {
			feed.Title = item.Description
			continue
		}

		if item.Type == gopher.INFO {
			continue
		}

		if rss_item_count >= MAX_RSS_FEED_ITEMS {
			break
		}

		if strings.HasPrefix(item.Selector, "URL:") {
			url := item.Selector[4:]
			fmt.Printf("url type one %s\n", url)
		} else {
			var hostport string
			if item.Port == 70 {
				hostport = item.Host
			} else {
				hostport = fmt.Sprintf("%s:%d", item.Host, item.Port)
			}

			path := url.PathEscape(item.Selector)
			path = strings.Replace(path, "%2F", "/", -1)

			url := fmt.Sprintf("%s/%s%s", hostport, string(byte(item.Type)), path)

			new_item, err := newFeedItem(item.Description, url)
			if err != nil {
				fmt.Printf("error... %s", err)
				continue
			}

			feed.Items = append(feed.Items, new_item)
			rss_item_count += 1
		}
	}

	rss, err := feed.ToRss()
	if err != nil {
		fmt.Printf("error making feed into rss %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	}

	w.Write([]byte(rss))

	return nil
}

func gopherHandler(uri string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {

		path := strings.TrimPrefix(req.URL.Path, "/")

		parts := strings.Split(path, "/")
		hostport := parts[0]

		if len(hostport) == 0 {
			http.Redirect(w, req, "/"+uri, http.StatusFound)
			return
		}

		var query_string string

		if req.URL.RawQuery != "" {
			query_string = fmt.Sprintf("?%s", url.QueryEscape(req.URL.RawQuery))
		}

		uri, err := url.QueryUnescape(strings.Join(parts[1:], "/"))
		if err != nil {
			io.WriteString(w, fmt.Sprintf("Error: %s", err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		res, err := gopher.Get(
			fmt.Sprintf(
				"gopher://%s/%s%s",
				hostport,
				uri,
				query_string,
			),
		)
		if err != nil {
			io.WriteString(w, fmt.Sprintf("Error: %s", err))
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		if res.Body != nil {
			// TODO: this is not a directory... meaning it can't really be turned into an rss feed.
			// throw an error?
			io.WriteString(w, "This is not a directory, cannot be turned into an rss feed!")
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		} else {
			if err := renderRss(w, hostport, res.Dir); err != nil {
				io.WriteString(w, fmt.Sprintf("Error: %s", err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}
}

func listenAndServe(bind, uri string) error {
	http.HandleFunc("/", gopherHandler(uri))
	return http.ListenAndServe(bind, nil)
}
