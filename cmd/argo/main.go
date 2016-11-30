// Copyright 2016 The argo Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/sbinet/argo"
)

var (
	mode = flag.String("mode", "led", "argo mode (led|sensor)")
	baud = flag.Int("baud", 57600, "baud rate")
)

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
		for data := range bot.Data {
			fmt.Printf("raw=%8.3f %v\n", data.Data, data.Time)
		}
	}()

	err = bot.Start()
	log.Fatal(err)
}
