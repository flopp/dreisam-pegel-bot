package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flopp/dreisam-pegel-bot/internal/pegel"
	"github.com/fogleman/gg"
	mastodon "github.com/mattn/go-mastodon"
)

func readMastodonConfig(fileName string) (mastodon.Config, error) {
	confBytes, err := os.ReadFile(fileName)
	if err != nil {
		return mastodon.Config{}, fmt.Errorf("reading %s: %s", fileName, err)
	}

	var conf mastodon.Config
	if err = json.Unmarshal(confBytes, &conf); err != nil {
		return mastodon.Config{}, fmt.Errorf("unmarshalling %s: %s", fileName, err)
	}

	return conf, nil
}

func startOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func drawBoxed(dc *gg.Context, s string, x, y float64, center bool) {
	w, h := dc.MeasureString(s)

	if center {
		x -= w / 2
	}

	dc.DrawRectangle(x-1, y-h-1, w+2, h+2)
	dc.SetHexColor("#FFFFFF")
	dc.Fill()

	dc.SetHexColor("#000000")
	dc.DrawString(s, x, y)
}

func renderChart(data pegel.PegelData) ([]byte, error) {
	var min int64 = data.Pegel.Value
	var max int64 = min
	for _, d := range data.Chart {
		if d.Value < min {
			min = d.Value
		} else if d.Value > max {
			max = d.Value
		}
	}
	fmt.Println(data.Chart[0].TimeStamp, data.Chart[len(data.Chart)-1].TimeStamp)
	fmt.Println(min, max)

	w := 7 * 24 * 4
	h := 200
	dc := gg.NewContext(w, h)
	dc.SetHexColor("#FFFFFF")
	dc.Clear()

	oneWeekAgo := data.Pegel.TimeStamp.Add(-7 * 24 * time.Hour)
	for _, d := range data.Chart {
		x := d.TimeStamp.Sub(oneWeekAgo).Hours() * 4
		y := float64(d.Value)

		dc.DrawLine(x, float64(h), x, float64(h)-y)
		dc.SetHexColor("#0000FF")
		dc.Stroke()
	}

	today := startOfDay(data.Pegel.TimeStamp)
	for d := 0; d < 7; d += 1 {
		dd := today.AddDate(0, 0, -d)
		x := float64(dd.Sub(oneWeekAgo).Hours() * 4)
		dc.DrawLine(x, 0, x, float64(h))
		dc.SetHexColor("#000000")
		dc.Stroke()

		drawBoxed(dc, dd.Format("2006-01-02"), x, float64(h-2), true)
	}

	for y := 105; y <= 145; y += 20 {
		dc.DrawLine(0, float64(h-y), float64(w), float64(h-y))
		dc.SetHexColor("#FF0000")
		dc.Stroke()

		drawBoxed(dc, fmt.Sprintf("%dcm", y), 0+1, float64(h-y+2), false)
	}

	drawBoxed(dc, fmt.Sprintf("Dreisam-Pegel: %dcm (%s)", data.Pegel.Value, data.Pegel.TimeStamp.Format(pegel.TimeLayout)), 1, 14, false)

	buff := new(bytes.Buffer)
	err := dc.EncodePNG(buff)
	if err != nil {
		return nil, fmt.Errorf("failed to enode png: %w", err)
	}

	return buff.Bytes(), nil
}

func createMessage(data pegel.PegelData) (string, bool) {
	tags := make([]string, 0)
	tags = append(tags, "#freiburg", "#dreisam")

	message := fmt.Sprintf("Dreisam-Pegel: %dcm (%s)\n", data.Pegel.Value, data.Pegel.TimeStamp.Format("2006-01-02 15:04"))

	trend := ""
	for _, t := range data.Trend {
		if t > 0 {
			trend += "⬆️"
		} else if t < 0 {
			trend += "⬇️"
		} else {
			trend += "⏺"
		}
	}
	if len(trend) > 0 {
		message += fmt.Sprintf("Trend: %s\n", trend)
	}

	stufe := 0
	if data.Pegel.Value >= 145 {
		stufe = 3
	} else if data.Pegel.Value >= 125 {
		stufe = 2
	} else if data.Pegel.Value >= 105 {
		stufe = 1
	}
	if stufe > 0 {
		message += fmt.Sprintf("\n⚠️ Achtung: Sperrstufe %d\nhttps://www.freiburg.de/pb/411886.html", stufe)
		tags = append(tags, "#fr1", "#hochwasser")
	}

	message += "\n\n"
	message += strings.Join(tags, " ")

	return message, stufe > 0
}

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("USAGE: %s <config-file> <data-dir>\n", os.Args[0])
		os.Exit(1)
	}

	configFile := os.Args[1]
	dataDir := os.Args[2]

	// update & get data
	data, err := pegel.GetPegelData(dataDir)
	if err != nil {
		panic(fmt.Errorf("failed to get pegel data: %w", err))
	}

	message, warning := createMessage(data)

	// post schedule:
	// regular messages: 12:00
	// warning messages: 00:00, 06:00, 12:00, 18:00
	now := time.Now().Local()
	if now.Minute() >= 15 {
		return
	}
	if warning {
		if now.Hour() != 0 && now.Hour() != 6 && now.Hour() != 12 && now.Hour() != 18 {
			return
		}
	} else {
		if now.Hour() != 12 {
			return
		}
	}

	mastodonConfig, err := readMastodonConfig(configFile)
	if err != nil {
		panic(fmt.Errorf("failed to get read mastodon config: %w", err))
	}

	client := mastodon.NewClient(&mastodonConfig)
	ctx := context.Background()

	toot := &mastodon.Toot{
		Status:     message,
		Visibility: "unlisted",
	}

	if chart, err := renderChart(data); err == nil {
		if attachment, err := client.UploadMediaFromBytes(ctx, chart); err == nil {
			toot.MediaIDs = append(toot.MediaIDs, attachment.ID)
		} else {
			fmt.Printf("Cannot upload attachment: %v\n", err)
		}
	} else {
		fmt.Printf("Cannot create chart: %v\n", err)
	}

	if _, err := client.PostStatus(context.Background(), toot); err != nil {
		panic(fmt.Errorf("failed to send status: %w", err))
	}
}
