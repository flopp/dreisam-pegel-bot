# dreisam-pegel-bot

This is a Mastodon bot that regularly posts information about the current water level of the Dreisam river in Freiburg/Germany.

https://freiburg.social/@dreisampegel

## How it works:

1. The bot is activated every couple of hours by a cronjob
2. It fetches the current water level data from the official water level site (https://www.hvz.baden-wuerttemberg.de/pegel.html?id=00389) and stores it into a timeseries CSV file 
4. If the water level is critically high (>105cm, at which there are closures of nearby bike lanes) or the last post was > 1 day ago, it will contine
5. A chart is rendered from the CSV file using github.com/fogleman/gg
6. A Mastodon post is made using github.com/mattn/go-mastodon, which includes the rendered chart, was well as some additional information

## Some Images:

The "normal" chart (no critical water level); current time is on the right, 1 week history to the left:

![Normal chart](/images/pegel.png)

The "critical" chart (water level > 105cm); note that there are three official criticality levels (105cm, 125cm, 145cm) at which the official perform certain actions (e.g. closure of paths along the river, etc.) - they are marked as horizontal lines.  

![Chart with critical water level](/images/pegel-critical.png)

The actual measuring station near Freiburg-Ebnet:

![The actual measuring station](/images/dreisam.jpg)
