//go:build ledglasses_nrf52840

// This example demostrates how to control the "Neopixel" (WS2812) LED included
// on the Adafruit LED Glasses Driver.  It implements a "rainbow effect" based
// on the following example:
// https://github.com/adafruit/Adafruit_Learning_System_Guides/blob/master/CircuitPython_Essentials/CircuitPython_Internal_RGB_LED_rainbow.py
package main

import (
	"context"
	"image/color"
	"machine"
	"strconv"
	"time"

	"tinygo.org/x/drivers/lis3dh"
	"tinygo.org/x/drivers/ws2812"
)

var (
	neo ws2812.Device
	on  = true

	accel = lis3dh.New(i2c)

	i2c   = machine.I2C0
	pwm   = machine.PWM0
	leds  = make([]color.RGBA, 1)
	wheel = &Wheel{Brightness: 0x02}

	button = machine.BUTTON
)

const (
	stateIdle = iota
	statePressed
)

func init() {

	machine.InitADC()

	machine.NEOPIXEL.Configure(machine.PinConfig{Mode: machine.PinOutput})
	neo = ws2812.New(machine.NEOPIXEL)

	// initialize i2c bus
	if err := i2c.Configure(machine.I2CConfig{
		SCL:       machine.SCL_PIN,
		SDA:       machine.SDA_PIN,
		Frequency: 400 * machine.KHz,
	}); err != nil {
		for {
			println("error: could not initialize I2C bus: " + err.Error())
			time.Sleep(time.Second)
		}
	}

	button.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	// Configure the regular on-board LED for PWM fading
	err := pwm.Configure(machine.PWMConfig{})
	if err != nil {
		println("failed to configure PWM")
		return
	}
}

func main() {

	time.Sleep(time.Second)

	// println("initializing bluetooth")
	// adapter := bluetooth.DefaultAdapter
	// if err := adapter.Enable(); err != nil {
	// 	for {
	// 		println("error: " + err.Error())
	// 		time.Sleep(time.Second)
	// 	}
	// }

	// We'll fade the on-board LED in a goroutine to ensure that the peripherals
	// works fine with the scheduler enabled.
	go func() {
		channelLED, err := pwm.Channel(machine.LED)
		if err != nil {
			println("failed to configure LED PWM channel")
			return
		}
		for i, brightening := uint8(0), false; ; i++ {
			if i == 0 {
				brightening = !brightening
				continue
			}
			var brightness uint32 = uint32(i)
			if !brightening {
				brightness = 256 - brightness
			}
			pwm.Set(channelLED, pwm.Top()*brightness/256)
			time.Sleep(5 * time.Millisecond)
		}
	}()

	go func() {
		vdiv := machine.ADC{Pin: machine.A6}
		vdiv.Configure(machine.ADCConfig{})
		for {
			val := vdiv.Get() >> 4
			// vbat := (float32(val*2) / float32(0xFFFF>>4)) * 3.3
			vbat := int(val) * 2000 / 1241 // adcValue * 2000 / (4095 / 3.3)
			println(
				"\rVBat:", strconv.Itoa(int(vbat)),
				// "\rVBat:", strconv.FormatFloat(float64(vbat), 'f', 6, 32), ";",
				// "ADC measurement:", strconv.FormatUint(uint64(val), 2), strconv.FormatUint(uint64(val), 10),
				"                    ")
			// float32 = analogRead(VBATPIN)
			// measuredvbat *= 2    // we divided by 2, so multiply back
			// measuredvbat *= 3.3  // Multiply by 3.3V, our reference voltage
			// measuredvbat /= 1024 // convert to voltage
			// Serial.print("VBat: ")
			// Serial.println(measuredvbat)
			time.Sleep(time.Second)
		}
	}()

	go func() {
		accel.Address = lis3dh.Address0 // address on the LED Glasses Driver
		accel.Configure()
		accel.SetRange(lis3dh.RANGE_2_G)
		println(accel.Connected())
		for {
			if connected := accel.Connected(); connected {
				x, y, z, _ := accel.ReadAcceleration()
				print("\r  X:", x, "  Y:", y, "  Z:", z, "        \r")
			} else {
				print("warning: LIS3DH not connected")
			}
			time.Sleep(time.Millisecond * 100)
		}
	}()

	// callback function to toggle the Neopixel on and off when the button is pressed
	onButtonPress := func(state int) {
		switch state {
		case statePressed:
			on = !on
			println("neopixel on:", on, "                           ")
		}
	}
	go debounceButton(context.Background(), onButtonPress)
	// go bleuart()
	// Use the "wheel" function from Adafruit's example to cycle the Neopixel
	for {
		leds[0] = wheel.Next()
		// println(leds[0].R, leds[0].G, leds[0].B, leds[0].A)
		neo.WriteColors(leds)
		time.Sleep(25 * time.Millisecond)
	}

}

// Wheel is a port of Adafruit's Circuit Python example referenced above.
type Wheel struct {
	Brightness uint8
	pos        uint8
}

// Next increments the internal state of the color and returns the new RGBA
func (w *Wheel) Next() (c color.RGBA) {
	if !on {
		return color.RGBA{0, 0, 0, 0}
	}
	pos := w.pos
	if w.pos < 85 {
		c = color.RGBA{R: 0xFF - pos*3, G: pos * 3, B: 0x0, A: w.Brightness}
	} else if w.pos < 170 {
		pos -= 85
		c = color.RGBA{R: 0x0, G: 0xFF - pos*3, B: pos * 3, A: w.Brightness}
	} else {
		pos -= 170
		c = color.RGBA{R: pos * 3, G: 0x0, B: 0xFF - pos*3, A: w.Brightness}
	}
	w.pos++
	return
}

// debounce presses for the button/switch on the board
func debounceButton(ctx context.Context, callback func(state int)) {
	state := stateIdle
	debounce := uint8(0)
	transitionTo := func(newState int) {
		if newState != state {
			debounce = 0
			state = newState
			if callback != nil {
				callback(state)
			} else {
				println("warning: no callback defined; transitioned to ", state)
			}
		}
	}
	readSwitch := func() {
		debounce <<= 1
		if !button.Get() {
			debounce |= 1
		}
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				switch state {
				case stateIdle:
					readSwitch()
					if debounce == 0xFF {
						transitionTo(statePressed)
					}
				case statePressed:
					readSwitch()
					if debounce == 0x00 {
						transitionTo(stateIdle)
					}
				}
			}
			time.Sleep(1 * time.Millisecond)
		}
	}()
}
