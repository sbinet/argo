// Copyright 2016 The argo Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/color"
	"log"
	"net"
	"net/http"

	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/vg"
	"github.com/gonum/plot/vg/draw"
	"github.com/gonum/plot/vg/vgsvg"
	"github.com/sbinet/argo"
	"golang.org/x/net/websocket"
)

var (
	mode  = flag.String("mode", "led", "argo mode (led|sensor)")
	baud  = flag.Int("baud", 57600, "baud rate")
	addr  = flag.String("addr", ":8080", "address:port of web server")
	datac = make(chan Data)
)

type Data struct {
	Plot string `json:"plot"`
}

func main() {
	flag.Parse()

	var m argo.Mode
	switch *mode {
	case "led":
		m = argo.LED
	case "sensor":
		m = argo.Sensor
	default:
		log.Fatalf("invalid mode (got=%q. want=%q|%q)\n", *mode, "led", "sensor")
	}

	bot, err := argo.New(m, "/dev/ttyACM0", *baud)
	if err != nil {
		log.Fatalf("error creating argo bot: %v\n", err)
	}
	defer bot.Stop()

	go func() {
		err = bot.Start()
		log.Fatal(err)
	}()

	go func() {
		var table = make(plotter.XYs, 0, 1024)

		for data := range bot.Data {
			x := float64(data.Time.UTC().Unix())
			y := data.Data
			table = append(table, struct{ X, Y float64 }{})
			i := len(table) - 1
			table[i].X = x
			table[i].Y = y
			datac <- Data{plotTime(table)}
			if len(table) == cap(table) {
				copy(table[:512], table[512:])
				table = table[:512]
			}
		}
	}()

	host, port, err := net.SplitHostPort(*addr)
	if err != nil {
		log.Fatal(err)
	}
	if host == "" {
		host = "localhost"
	}
	log.Printf("please connect to: http://%s:%s\n", host, port)
	http.HandleFunc("/", plotHandle)
	http.Handle("/data", websocket.Handler(dataHandler))
	log.Fatal(http.ListenAndServe(host+":"+port, nil))
}

func plotTime(data plotter.XYs) string {
	p, err := plot.New()
	if err != nil {
		log.Fatalf("error creating time-plot: %v\n", err)
	}

	// xticks defines how we convert and display time.Time values.
	xticks := plot.TimeTicks{Format: "2006-01-02\n15:04:05"}

	p.X.Label.Text = "Time"
	p.X.Tick.Marker = xticks
	p.Y.Min = 0
	p.Y.Label.Text = "Light (A.U.)"

	line, points, err := plotter.NewLinePoints(data)
	if err != nil {
		log.Fatal(err)
	}

	line.LineStyle.Color = color.RGBA{255, 0, 0, 255}
	line.LineStyle.Width = vg.Points(1)
	points.Shape = draw.CircleGlyph{}
	points.Color = color.RGBA{R: 255, A: 255}

	p.Add(points)
	if false {
		p.Add(line)
	}
	p.Add(plotter.NewGrid())

	return renderSVG(p)
}

func renderSVG(p *plot.Plot) string {
	size := 20 * vg.Centimeter
	canvas := vgsvg.New(2*size, size)
	p.Draw(draw.New(canvas))
	out := new(bytes.Buffer)
	_, err := canvas.WriteTo(out)
	if err != nil {
		panic(err)
	}
	return string(out.Bytes())
}

func plotHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, page)
}

func dataHandler(ws *websocket.Conn) {
	for data := range datac {
		err := websocket.JSON.Send(ws, data)
		if err != nil {
			log.Printf("error sending data: %v\n", err)
			return
		}
	}
}

const page = `
<html>
	<head>
		<title>Plotting stuff with gonum/plot</title>
		<script type="text/javascript">
		var sock = null;
		var plot = "";

		function update() {
			var p = document.getElementById("my-plot");
			p.innerHTML = plot;
		};

		window.onload = function() {
			sock = new WebSocket("ws://"+location.host+"/data");

			sock.onmessage = function(event) {
				var data = JSON.parse(event.data);
				//console.log("data: "+JSON.stringify(data));
				plot = data.plot;
				update();
			};
		};

		</script>

		<style>
		.my-plot-style {
			width: 400px;
			height: 200px;
			font-size: 14px;
			line-height: 1.2em;
		}
		</style>
	</head>

	<body>
		<div id="header">
			<h2>My plot</h2>
		</div>

		<div id="content">
			<div id="my-plot" class="my-plot-style"></div>
		</div>
	</body>
</html>
`
