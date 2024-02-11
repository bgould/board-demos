package main

// This example implements a NUS (Nordic UART Service) peripheral.
// I can't find much official documentation on the protocol, but this can be
// helpful:
// https://learn.adafruit.com/introducing-adafruit-ble-bluetooth-low-energy-friend/uart-service
//
// Code to interact with a raw terminal is in separate files with build tags.

import (
	"bufio"
	"machine"
	"strconv"
	"time"

	"tinygo.org/x/bluetooth"
)

var (
	serviceUUID = bluetooth.ServiceUUIDNordicUART
	rxUUID      = bluetooth.CharacteristicUUIDUARTRX
	txUUID      = bluetooth.CharacteristicUUIDUARTTX

	console = NewConsole(machine.Serial)

	rxChar bluetooth.Characteristic
	txChar bluetooth.Characteristic

	buf = machine.NewRingBuffer()

	out = bufio.NewWriter(console)
)

func main() {
	time.Sleep(3 * time.Second)

	println("starting")
	adapter := bluetooth.DefaultAdapter
	must("enable BLE stack", adapter.Enable())
	adv := adapter.DefaultAdvertisement()
	must("config adv", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName:    "NUS", // Nordic UART Service
		ServiceUUIDs: []bluetooth.UUID{serviceUUID},
	}))
	must("start adv", adv.Start())

	must("add service", adapter.AddService(&bluetooth.Service{
		UUID: serviceUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &rxChar,
				UUID:   rxUUID,
				Flags:  bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					for _, b := range value {
						buf.Put(b)
					}
					// txChar.Write(value)
					// console.WriteString(string(value))
					// for _, c := range value {
					// rawterm.Putchar(c)
					// }

				},
			},
			{
				Handle: &txChar,
				UUID:   txUUID,
				Flags:  bluetooth.CharacteristicNotifyPermission | bluetooth.CharacteristicReadPermission,
			},
		},
	}))

	go readBattery()

	for {
		for {
			b, ok := buf.Get()
			if !ok {
				out.Flush()
				break
			}
			out.WriteByte(b)
		}
		console.Task()
		// runtime.Gosched()
		time.Sleep(1 * time.Millisecond)
	}

	// rawterm.Configure()
	// defer rawterm.Restore()
	// print("NUS console enabled, use Ctrl-X to exit\r\n")
	// var line []byte
	// for {
	// 	ch := rawterm.Getchar()
	// 	rawterm.Putchar(ch)
	// 	line = append(line, ch)

	// 	// Send the current line to the central.
	// 	if ch == '\x18' {
	// 		// The user pressed Ctrl-X, exit the terminal.
	// 		break
	// 	} else if ch == '\n' {
	// 		sendbuf := line // copy buffer
	// 		// Reset the slice while keeping the buffer in place.
	// 		line = line[:0]

	// 		// Send the sendbuf after breaking it up in pieces.
	// 		for len(sendbuf) != 0 {
	// 			// Chop off up to 20 bytes from the sendbuf.
	// 			partlen := 20
	// 			if len(sendbuf) < 20 {
	// 				partlen = len(sendbuf)
	// 			}
	// 			part := sendbuf[:partlen]
	// 			sendbuf = sendbuf[partlen:]
	// 			// This also sends a notification.
	// 			_, err := txChar.Write(part)
	// 			must("send notification", err)
	// 		}
	// 	}
	// }
}

func must(action string, err error) {
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}

func readBattery() {
	vdiv := machine.ADC{Pin: machine.P0_29}
	vdiv.Configure(machine.ADCConfig{})
	for {
		val := vdiv.Get() >> 4
		// vbat := (float32(val*2) / float32(0xFFFF>>4)) * 3.3
		vbat := int(val) * 2000 / 1241 // adcValue * 2000 / (4095 / 3.3)
		str := "VBat: " + strconv.Itoa(int(vbat)) + "\n"
		console.WriteString(str)
		txChar.Write([]byte(str))
		// "\rVBat:", strconv.FormatFloat(float64(vbat), 'f', 6, 32), ";",
		// "ADC measurement:", strconv.FormatUint(uint64(val), 2), strconv.FormatUint(uint64(val), 10),
		// float32 = analogRead(VBATPIN)
		// measuredvbat *= 2    // we divided by 2, so multiply back
		// measuredvbat *= 3.3  // Multiply by 3.3V, our reference voltage
		// measuredvbat /= 1024 // convert to voltage
		// Serial.print("VBat: ")
		// Serial.println(measuredvbat)
		time.Sleep(time.Second)
	}
}
