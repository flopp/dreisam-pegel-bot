package main

import (
	"fmt"
	"os"

	"github.com/flopp/dreisam-pegel-bot/internal/chart"
	"github.com/flopp/dreisam-pegel-bot/internal/pegel"
)

func main() {
	if len(os.Args) != 2 {
		panic("cache dir missing")
	}

	p, err := pegel.GetPegelData(os.Args[1])
	if err != nil {
		fmt.Println("cannot get pegel:", err)
	} else {
		fmt.Println(p.Pegel.TimeStamp.Format(pegel.TimeLayout))
		fmt.Println(p.Pegel.Value)
		for _, t := range p.Trend {
			fmt.Println(t)
		}

		if buf, err := chart.RenderChart(p); err != nil {
			panic(err)
		} else {
			if err := os.WriteFile("chart.png", buf, 0o666); err != nil {
				panic(err)
			}
		}
	}
}
