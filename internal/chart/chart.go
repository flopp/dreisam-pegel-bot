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

func drawBar(dc *gg.Context, h int, x int, value int) {
	y := 2 * value
	dc.DrawLine(float64(x), float64(h), float64(x), float64(h-y))
	if value >= 105 {
		dc.SetHexColor("#FF0000")
	} else {
		dc.SetHexColor("#0000FF")
	}
	dc.Stroke()

}

func RenderChart(data pegel.PegelData) ([]byte, error) {
	w := 7 * 24 * 4
	h := 400
	dc := gg.NewContext(w, h)
	dc.SetHexColor("#FFFFFF")
	dc.Clear()

	oneWeekAgo := data.Pegel.TimeStamp.Add(-7 * 24 * time.Hour)
	last_x := 0
	last_v := -1
	for _, d := range data.Chart {
		x := int(d.TimeStamp.Sub(oneWeekAgo).Hours() * 4)

		// interpolate missing values
		if last_v >= 0 && x > last_x+1 {
			d_v := float64(int(d.Value)-last_v) / float64(x-last_x)
			for xx := last_x + 1; xx < x; xx += 1 {
				drawBar(dc, h, xx, last_v+int(float64(xx-last_x)*d_v))
			}
		}

		drawBar(dc, h, x, int(d.Value))
		last_x = x
		last_v = int(d.Value)
	}

	today := startOfDay(data.Pegel.TimeStamp)
	for d := 0; d < 7; d += 1 {
		dd := today.AddDate(0, 0, -d)
		x := float64(dd.Sub(oneWeekAgo).Hours() * 4)
		dc.DrawLine(x, 0, x, float64(h))
		dc.SetDash(2, 10)
		dc.SetHexColor("#000000")
		dc.Stroke()

		drawBoxed(dc, dd.Format("2006-01-02"), x, float64(h-2), true)
	}

	for limit := 105; limit <= 145; limit += 20 {
		y := 2 * limit

		dc.DrawLine(0, float64(h-y), float64(w), float64(h-y))
		dc.SetDash()
		dc.SetHexColor("#000000")
		dc.Stroke()

		drawBoxed(dc, fmt.Sprintf("%dcm (Sperrstufe %d)", limit, (limit-105)/20+1), 1, float64(h-y), false)
	}

	for _, i := range []int{25, 50, 75, 175} {
		y := 2 * i

		dc.DrawLine(0, float64(h-y), float64(w), float64(h-y))
		dc.SetDash(2, 10)
		dc.SetHexColor("#000000")
		dc.Stroke()

		drawBoxed(dc, fmt.Sprintf("%dcm", i), 1, float64(h-y), false)
	}

	drawBoxed(dc, fmt.Sprintf("Dreisam-Pegel: %dcm (%s)", data.Pegel.Value, data.Pegel.TimeStamp.Format(pegel.TimeLayout)), 1, 14, false)

	buff := new(bytes.Buffer)
	err := dc.EncodePNG(buff)
	if err != nil {
		return nil, fmt.Errorf("failed to enode png: %w", err)
	}

	return buff.Bytes(), nil
}
