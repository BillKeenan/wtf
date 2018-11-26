package deaddrop

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/rivo/tview"
	"github.com/senorprogrammer/wtf/logger"
	"github.com/senorprogrammer/wtf/wtf"
)

// Config is a pointer to the global config object

var started = false
var baseURL = "https://min-api.cryptocompare.com/data/price"
var ok = true
var starsCount = 0

const configKey = "deaddrop"

// Widget define wtf widget to register widget later
type Widget struct {
	wtf.BarGraph

	// time interval for send http request
	updateInterval int
}

// NewWidget Make new instance of widget
func NewWidget(app *tview.Application) *Widget {
	widget := Widget{
		BarGraph: wtf.NewBarGraph(" Dead Drop ", configKey, false),
	}

	widget.View.SetWrap(true)
	widget.View.SetWordWrap(true)

	return &widget
}

/* -------------------- Exported Functions -------------------- */

// GetDrops - Load the dead drop stats
func GetDrops(widget *Widget) {
	logger.Log("loading dead drop data")

	var (
		client http.Client
	)

	client = http.Client{
		Timeout: time.Duration(5 * time.Second),
	}

	req, err := http.NewRequest("GET", "https://dead-drop.me/stats/json", nil)
	if err != nil {
		widget.View.SetText(fmt.Sprintf("%s", err.Error()))
		return
	}
	req.Header.Set("User-Agent", "curl")
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

	type dropsStruct []struct {
		Data  [][]interface{} `json:"data"`
		Label string          `json:"label"`
	}

	var drops dropsStruct

	err2 := json.Unmarshal(contents, &drops)

	if err2 != nil {
		fmt.Println("error:", err)
	}

	var s = drops[0].Data

	const lineCount = 20

	var stats [lineCount][2]int64

	var count = 0
	for i := len(s) - 1; i >= len(s)-lineCount; i-- {
		var val = s[i][1].(float64)
		stats[count][0] = int64(val)
		stats[count][1] = int64(s[i][0].(float64))
		count++
	}

	logger.Log("loaded dead data")
	widget.View.Clear()
	widget.BuildBars(stats[:])

}

// Refresh & update after interval time
func (widget *Widget) Refresh() {

	if widget.Disabled() {
		return
	}

	if !ok {
		widget.View.SetText(
			fmt.Sprint("Please check your internet connection!"),
		)
		return
	}

	display(widget)

}

/* -------------------- Unexported Functions -------------------- */

func display(widget *Widget) {
	GetDrops(widget)
	widget.View.SetTitle("☠️ " + widget.Name)
}
