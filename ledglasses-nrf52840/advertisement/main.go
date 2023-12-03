package main

import (
	"time"

	"tinygo.org/x/bluetooth"
)

var adapter = bluetooth.DefaultAdapter

func main() {
	time.Sleep(time.Second)

	println("Enabling stack in 3 .. 2 .. 1")

	time.Sleep(3 * time.Second)

	println("Enabling BLE stack")
	must("enable BLE stack", adapter.Enable())

	time.Sleep(1 * time.Second)
	println("advertising...")

	adv := adapter.DefaultAdvertisement()
	must("config adv", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName: "Go Bluetooth",
	}))
	must("start adv", adv.Start())

	address, _ := adapter.Address()
	for {
		println("Go Bluetooth /", address.MAC.String())
		// println("tick")
		time.Sleep(time.Second)
	}
}

func must(action string, err error) {
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}
