// Copyright 2016 The argo Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package argo

import (
	"log"
	"time"

	"github.com/hybridgroup/gobot"
	"github.com/hybridgroup/gobot/platforms/firmata"
	"github.com/hybridgroup/gobot/platforms/gpio"
	"github.com/tarm/goserial"
)

type Data struct {
	Time time.Time `json:"time"`
	Data float64   `json:"data"`
}

type Bot struct {
	Bot  *gobot.Gobot
	Data chan Data
}

func New(mode Mode, device string, baud int) (*Bot, error) {
	if device == "" {
		device = "/dev/ttyACM0"
	}
	if baud == 0 {
		baud = 57600
	}

	bot := &Bot{
		Bot:  gobot.NewGobot(),
		Data: make(chan Data),
	}

	log.Printf("--> device: %q baud: %d\n", device, baud)

	conn, err := serial.OpenPort(&serial.Config{Name: device, Baud: baud})
	if err != nil {
		return nil, err
	}

	fa := firmata.NewFirmataAdaptor("arduino", device, conn)
	led := gpio.NewLedDriver(fa, "led", "13")
	sensor := gpio.NewAnalogSensorDriver(fa, "sensor", "1", 500*time.Millisecond)

	var work func()

	switch mode {
	case LED:
		work = func() {
			gobot.Every(1*time.Second, func() {
				led.Toggle()
			})
		}
	case Sensor:
		work = func() {
			gobot.Every(1*time.Second, func() {
				led.Toggle()
			})
			sensor.On(gpio.Data, func(data interface{}) {
				v := float64(data.(int))
				select {
				case bot.Data <- Data{Time: time.Now(), Data: v}:
				default: // nobody's listening, drop it
				}
			})
			sensor.On(gpio.Error, func(data interface{}) {
				log.Printf("error: %v\n", data)
			})
		}
	default:
		log.Fatal("invalid mode (got=%q. want=%q|%q)\n", mode, "led", "sensor")
	}

	bot.Bot.AddRobot(
		gobot.NewRobot("bot",
			[]gobot.Connection{fa},
			[]gobot.Device{led, sensor},
			work,
		),
	)

	return bot, nil
}

func (bot *Bot) Start() error {
	errs := bot.Bot.Start()
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (bot *Bot) Stop() error {
	errs := bot.Bot.Stop()
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

type Mode int

const (
	LED Mode = iota
	Sensor
)
