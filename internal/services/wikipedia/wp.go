package wikipedia

import (
	"fmt"

	gowiki "github.com/trietmn/go-wiki"
)

func P() {
	gowiki.SetLanguage("ru")
	gowiki.SetUserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")
	// Search
	results, suggestion, err := gowiki.Search("Golang", 5, true)

	// Get detailed page
	page, err := gowiki.GetPage("Go (programming language)", -1, false, true)
	fmt.Println(err)
	content, err := page.GetContent()
	fmt.Println(err)
	fmt.Println(results)
	fmt.Println(suggestion)
	fmt.Println(content)
}

func Setgowiki(lang string) {
	if lang != "" {
		gowiki.SetLanguage(lang)
	}
	gowiki.SetUserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")
}

func Search(req string, limit int, suggest bool) ([]string, string, error) {
	return gowiki.Search(req, limit, suggest)
}

func GeoSearch(latitude float32, longitude float32, radius float32, title string, limit int) ([]string, error) {
	return gowiki.GeoSearch(latitude, longitude, radius, title, limit)
}

func GetBacklinks(title string) ([]string, error) {
	return gowiki.GetBacklinks(title)
}

func Summary(title string, numsentence int, numchar int, suggest bool, redirect bool) (string, error) {
	return gowiki.Summary(title, numsentence, numchar, suggest, redirect)
}
