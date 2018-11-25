package confluence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/rivo/tview"
	"github.com/senorprogrammer/wtf/logger"
	"github.com/senorprogrammer/wtf/wtf"
	"golang.org/x/net/html"
)

var started = false

const HelpText = `
  Keyboard commands for Textfile:

    /: Show/hide this help window
    o: Open the text file in the operating system
`

type Widget struct {
	wtf.TextWidget

	// time interval for send http request
	updateInterval int
}

func NewWidget(app *tview.Application) *Widget {

	widget := Widget{
		TextWidget: wtf.NewTextWidget(app, " Confluence ", "confluence", true),
	}

	widget.View.SetWrap(true)
	widget.View.SetWordWrap(true)

	return &widget
}

/* -------------------- Exported Functions -------------------- */

func (widget *Widget) Refresh() {
	if widget.Disabled() {
		return
	}

	display(widget)
}

/* -------------------- Unexported Functions -------------------- */

// GetDrops - Load the dead drop stats
func GetRates(widget *Widget) {

	var (
		client http.Client
	)

	client = http.Client{
		Timeout: time.Duration(5 * time.Second),
	}

	req, err := http.NewRequest("GET", "https://api-fxpractice.oanda.com/v1/prices?accountId=101-002-8383426-001&instruments=USD_CAD", nil)
	if err != nil {
		widget.View.SetText(fmt.Sprintf("%s", err.Error()))
		return
	}
	req.Header.Set("User-Agent", "curl")
	req.Header.Set("Authorization", "Bearer 967bd8c1553b73854d25b8cdfa46af6b-bb3bb1bd4bfe55f3f89aa49def4bb261")
	response, err := client.Do(req)
	if err != nil {
		widget.View.SetText(fmt.Sprintf("%s", err.Error()))
		return
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		widget.View.SetText(fmt.Sprintf("%s", err.Error()))
		return
	}

	type rateStruct struct {
		Prices []struct {
			Instrument string    `json:"instrument"`
			Time       time.Time `json:"time"`
			Bid        float64   `json:"bid"`
			Ask        float64   `json:"ask"`
		} `json:"prices"`
	}

	var rates rateStruct

	err2 := json.Unmarshal(contents, &rates)

	if err2 != nil {
		fmt.Println("error:", err)
	}

	const lineCount = 20

	var buffer bytes.Buffer

	for i := len(rates.Prices) - 1; i >= 0; i-- {
		buffer.WriteString(fmt.Sprintf("%s - bid : [green]%f[white] ask: [red]%f[white] \n", rates.Prices[i].Instrument, rates.Prices[i].Bid, rates.Prices[i].Ask))
	}

	logger.Log("loaded oanda data")

	widget.View.Clear()
	widget.View.SetText(buffer.String())
}

func display(widget *Widget) {
	GetRates(widget)
}

// Helper function to pull the href attribute from a Token
func getHref(t html.Token) (ok bool, href string) {
	// Iterate over all of the Token's attributes until we find an "href"
	for _, a := range t.Attr {
		//fmt.Println("\nFound", a.Key, "\n")

		if a.Key == "data-inline-task-id" {
			href = t.Data
			ok = true
		}
	}

	// "bare" return will return the variables (ok, href) as defined in
	// the function definition
	return
}

// Extract all http** links from a given webpage
func crawl(url string, ch chan string, chFinished chan bool) {

	req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth("bkeenan@oanda.com", "HpxjOY1dpX5XHnCagXWFDFA2")

	req.Header.Set("Accept", "application/json")

	debug(httputil.DumpRequestOut(req, true))

	cli := &http.Client{}
	resp, err := cli.Do(req)

	defer func() {
		// Notify that we're done after this function
		chFinished <- true
	}()

	if err != nil {
		fmt.Println("ERROR: Failed to crawl \"" + url + "\"")
		return
	}

	b := resp.Body

	//bodyBytes, err := ioutil.ReadAll(resp.Body)
	//bodyString := string(bodyBytes)

	//fmt.Println("\nFound", bodyString, "\n")

	defer b.Close() // close Body when the function returns

	z := html.NewTokenizer(b)

	textState := false
	timeState := false
	nameState := false
	val := ""

	for {

		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return
		case tt == html.StartTagToken:
			t := z.Token()

			// Check if the token is an <a> tag
			isAnchor := t.Data == "li"

			if !isAnchor {

				if textState {

					switch t.Data {
					case "time":
						timeState = true
					case "a":
						for _, a := range t.Attr {
							if a.Key == "data-username" {
								nameState = true
							}
						}
					}
				}

				continue
			}

			textState = true

			// Extract the href value, if there is one
			ok, url := getHref(t)
			if !ok {
				continue
			}
			// Make sure the url begines in http**
			ch <- url
		case tt == html.TextToken:
			if textState {
				t := z.Token()
				if timeState {
					val += fmt.Sprintf("<%s>", strings.Trim(t.Data, "\n"))

				} else if nameState {
					val += fmt.Sprintf("|%s|", strings.Trim(t.Data, "\n"))

				} else {
					val += strings.Trim(t.Data, "\n")
				}
			}
		case tt == html.EndTagToken:
			if textState {
				t := z.Token()

				// Check if the token is an <a> tag
				if t.Data == "li" {
					textState = false
					fmt.Println(val)
					val = ""

				} else if t.Data == "time" {
					timeState = false
				} else if t.Data == "a" {
					nameState = false
				}

			}

		}

	}

}

func debug(data []byte, err error) {
	if err == nil {
		fmt.Printf("%s\n\n", data)
	} else {
		log.Fatalf("%s\n\n", err)
	}
}

func main2() {
	urlToGet := "https://oandacorp.atlassian.net/wiki/rest/api/content/698351720?expand=body.view"
	foundUrls := make(map[string]bool)
	seedUrls := os.Args[1:]

	// Channels
	chUrls := make(chan string)
	chFinished := make(chan bool)

	// Kick off the crawl process (concurrently)
	go crawl(urlToGet, chUrls, chFinished)

	// Subscribe to both channels
	for c := 0; c < len(seedUrls); {
		select {
		case url := <-chUrls:
			foundUrls[url] = true
		case <-chFinished:
			c++
		}
	}

	// We're done! Print the results...

	//fmt.Println("\nFound", len(foundUrls), "unique urls:\n")

	// for url := range foundUrls {
	// 	fmt.Println(" - " + url)
	// }

	close(chUrls)
}
