package main

import (
	"time"

	temperusb "github.com/gurupras/go-TEMPerUSB"
	log "github.com/sirupsen/logrus"
)

func main() {
	t, err := temperusb.New()
	if err != nil {
		log.Fatalf("Failed to get temperUSB: %v", err)
	}
	for {
		temp, err := t.GetTemperature()
		if err != nil {
			log.Errorf("Failed to get temperature: %v", err)
		}
		log.Infof("%.2f", temp)
		time.Sleep(300 * time.Millisecond)

	}
}
