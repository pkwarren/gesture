// Google Image Search functionality
package gis

import (
	"fmt"
	"github.com/collinvandyck/gesture/core"
	"github.com/collinvandyck/gesture/util"
	"math/rand"
	neturl "net/url"
	"strings"
	"time"
)

func Create(bot *core.Gobot, config map[string]interface{}) {
	defaultUrl, useDefault := config["default"].(string)
	exclusions := getExclusions(config)
	bot.ListenFor("^gis (.*)", func(msg core.Message, matches []string) core.Response {
		for _, ex := range(exclusions) {
			if ex == msg.Channel {
				return bot.Stop()
			}
		}
		link, err := search(matches[1])
		if err != nil {
			if useDefault {
				link = defaultUrl
			} else {
				return bot.Error(err)
			}
		}
		msg.Ftfy(link)
		return bot.Stop()
	})
}

func getExclusions(config map[string]interface{}) []string {
	result := make([]string, 0)
	exclude, ok := config["exclude"].([]interface{})
	if (!ok) {
		return result
	}
	for _, ex := range(exclude) {
		result = append(result, ex.(string))
	}
	return result
}


// these structs really tie the room together, man
type gisResult struct {
	Url string
}
type gisResponse struct {
	ResponseData struct {
		Results *[]gisResult // use a pointer here b/c sometimes the results are null :(
	}
}

// Search queries google for some images, and then randomly selects one
func search(search string) (string, error) {
	searchUrl := "http://ajax.googleapis.com/ajax/services/search/images?v=1.0&q=" + neturl.QueryEscape(search)
	var gisResponse gisResponse
	if err := util.UnmarshalUrl(searchUrl, &gisResponse); err != nil {
		return "", err
	}

	if gisResponse.ResponseData.Results == nil {
		return "", fmt.Errorf("No results were returned for query %s", search)
	}

	results := *gisResponse.ResponseData.Results

	if len(results) > 0 {

		// start a goroutine to determine image info for each response result
		// we have to use buffered channels so that the senders don't hang on send after the main method exits
		imageUrlCh := make(chan string, len(results))
		errorsCh := make(chan error, len(results))
		for _, resultUrl := range results {
			go getImageInfo(resultUrl.Url, imageUrlCh, errorsCh)
		}

		// until a timeout is met, build a collection of urls
		totalResults := len(results)
		remainingResults := totalResults
		urls := make([]string, 0, totalResults)
		errors := make([]error, 0, totalResults)
		timeout := time.After(2 * time.Second)

	SEARCH:
		for remainingResults > 0 {
			select {
			case url := <-imageUrlCh:
				urls = append(urls, url)
				remainingResults--
			case err := <-errorsCh:
				errors = append(errors, err)
				remainingResults--
			case <-timeout:
				break SEARCH
			}
		}
		if len(urls) == 0 {
			return "", fmt.Errorf("No image could be found for \"%s\"", search)
		}
		return urls[rand.Intn(len(urls))], nil

	}
	return "", fmt.Errorf("No image could be found for \"%s\"", search)
}

// getImageInfo looks at the header info for the url, and if it is an image, it sends an imageInfo on the channel
func getImageInfo(url string, ch chan<- string, failures chan<- error) {
	imageUrl, contentType, err := util.ResponseHeaderContentType(url)
	if err == nil && strings.HasPrefix(contentType, "image/") {
		url, err := ensureSuffix(imageUrl, "."+contentType[len("image/"):])
		if err != nil {
			failures <- err
		} else {
			ch <- url
		}
	} else {
		failures <- fmt.Errorf("Not an image: %s", url)
	}
}

// ensureSuffix ensures a url ends with suffixes like .jpg, .png, etc
func ensureSuffix(url, suffix string) (string, error) {
	var err error
	unescapedUrl, err := neturl.QueryUnescape(url)
	if err != nil {
		return "", err
	}
	lowerSuffix := strings.ToLower(suffix)
	lowerUrl := strings.ToLower(unescapedUrl)
	if lowerSuffix == ".jpeg" && strings.HasSuffix(lowerUrl, ".jpg") {
		return url, nil
	}
	if lowerSuffix == ".jpg" && strings.HasSuffix(lowerUrl, ".jpeg") {
		return url, nil
	}
	if strings.HasSuffix(lowerUrl, lowerSuffix) {
		return url, nil
	}
	if strings.Contains(url, "?") {
		return url + "&lol=lol" + suffix, nil
	}
	return url + "?lol=lol" + suffix, nil
}
