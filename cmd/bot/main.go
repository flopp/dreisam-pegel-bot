package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flopp/dreisam-pegel-bot/internal/chart"
	"github.com/flopp/dreisam-pegel-bot/internal/pegel"
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
	force := false
	if len(os.Args) == 3 {
		//
	} else if len(os.Args) == 4 && os.Args[3] == "-force" {
		force = true
	} else {
		fmt.Printf("USAGE: %s <config-file> <data-dir> [-force]\n", os.Args[0])
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

	if !force {
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

	if chart, err := chart.RenderChart(data); err == nil {
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
