package confluence

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"

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

	main2(widget)
}

/* -------------------- Unexported Functions -------------------- */
func display(widget *Widget) {
	logger.Log("in confluence")
	main2(widget)
}

// Extract all http** links from a given webpage
func crawl(widget *Widget, url string, user string, pass string) {

	req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(user, pass)

	req.Header.Set("Accept", "application/json")

	//debug(httputil.DumpRequestOut(req, true))

	cli := &http.Client{}
	resp, err := cli.Do(req)

	if err != nil {
		fmt.Println("ERROR: Failed to crawl \"" + url + "\"")
		return
	}

	b := resp.Body

	//body, err := ioutil.ReadAll(b)

	//logger.Log(string(body))
	defer b.Close() // close Body when the function returns

	z := html.NewTokenizer(b)

	textState := false
	timeState := false
	nameState := false
	val := ""

	var buffer bytes.Buffer

	for {

		tt := z.Next()

		if tt == html.ErrorToken {
			break
		}
		switch {
		case tt == html.StartTagToken:
			t := z.Token()

			// Check if the token is an <a> tag
			isLI := t.Data == "li"

			isTask := false

			for _, a := range t.Attr {
				if a.Key == "data-inline-task-id" {
					isTask = true
					break
				}
			}

			if !isLI {

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

			if !isTask {
				break
			}

			textState = true

			// Extract the href value, if there is one
			// Make sure the url begines in http**
		case tt == html.TextToken:
			if textState {
				t := z.Token()
				if timeState {
					val += fmt.Sprintf("[green] %s", strings.Trim(t.Data, "\n"))

				} else if nameState {
					val += fmt.Sprintf("[red] %s", strings.Trim(t.Data, "\n"))

				} else {
					val += fmt.Sprintf("[white] %s", strings.Trim(t.Data, "\n"))
				}
			}
		case tt == html.EndTagToken:
			if textState {
				t := z.Token()

				// Check if the token is an <a> tag
				if t.Data == "li" {
					textState = false

					buffer.WriteString(val)
					buffer.WriteString("\n")

					val = ""

				} else if t.Data == "time" {
					timeState = false
				} else if t.Data == "a" {
					nameState = false
				}

			}

		}

	}

	widget.View.Clear()
	widget.View.SetText(buffer.String())

	logger.Log("loaded confluence data")
}

func debug(data []byte, err error) {
	if err == nil {
		fmt.Printf("%s\n\n", data)
	} else {
		log.Fatalf("%s\n\n", err)
	}
}

func main2(widget *Widget) {
	logger.Log("getting confluence data")

	urlToGet := wtf.Config.UString("wtf.mods.confluence.url", "")
	user := wtf.Config.UString("wtf.mods.confluence.user", "")
	pass := wtf.Config.UString("wtf.mods.confluence.pass", "")

	crawl(widget, urlToGet, user, pass)

}
