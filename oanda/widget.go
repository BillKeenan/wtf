package oanda

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/senorprogrammer/wtf/logger"
	"github.com/senorprogrammer/wtf/wtf"
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

func NewWidget() *Widget {
	widget := Widget{
		TextWidget: wtf.NewTextWidget(" Oanda ", "oanda", true),
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

	widget.UpdateRefreshedAt()

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
