package chart

import (
	"bytes"
	"fmt"
	"time"

	"github.com/flopp/dreisam-pegel-bot/internal/pegel"
	"github.com/fogleman/gg"
)

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

func RenderChart(data pegel.PegelData) ([]byte, error) {
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
