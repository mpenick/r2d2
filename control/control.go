package control

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/howeyc/crc16"
	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
	"log"
	"strings"
	"time"
)

const robotID = "C5:DD:FB:6A:06:9E"
const robotService = "d9d9e9e0-aa4e-4797-8151-cb41cedaf2ad"
const robotBitsnap = "Bitsnap Control"

type Move int

const (
	Forward Move = iota
	Backward
	Left
	Right
)

type Control struct {
	cmds           chan Move
	peripheral     gatt.Peripheral
	characteristic *gatt.Characteristic
}

func NewControl() (*Control, error) {
	device, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to open device: %v", err)
	}

	done := make(chan struct {
		*Control
		error
	})

	device.Handle(
		gatt.PeripheralDiscovered(func(p gatt.Peripheral, advertisement *gatt.Advertisement, rssi int) {
			if strings.ToUpper(p.ID()) == robotID {
				log.Println("found robot")
				p.Device().StopScanning()
				p.Device().Connect(p)
			}
		}),
		gatt.PeripheralConnected(func(p gatt.Peripheral, err error) {
			ctrl := &Control{
				cmds:           make(chan Move, 5),
				peripheral:     p,
				characteristic: findBitsnapControl(p),
			}

			done <- struct {
				*Control
				error
			}{
				Control: ctrl,
				error:   nil,
			}

			ctrl.red(1)
			ctrl.reset()

			for move := range ctrl.cmds {
				switch move {
				case Forward:
					ctrl.motor2(0x89 + 59)
					<-time.After(2 * time.Second)
				case Backward:
					ctrl.motor2(0x89 - 59)
					<-time.After(2 * time.Second)
				case Left:
					ctrl.servo(0x7F + 64)
					<-time.After(1 * time.Second)
					ctrl.motor2(0x89 + 59)
					<-time.After(1 * time.Second)
				case Right:
					ctrl.servo(0x7F - 64)
					<-time.After(1 * time.Second)
					ctrl.motor2(0x89 + 59)
					<-time.After(1 * time.Second)
				}
				ctrl.reset()
			}
		}),
		gatt.PeripheralDisconnected(func(p gatt.Peripheral, err error) {
		}),
	)

	err = device.Init(func(device gatt.Device, state gatt.State) {
		switch state {
		case gatt.StatePoweredOn:
			device.Scan([]gatt.UUID{}, false)
			log.Println("looking for robot...")
			return
		default:
			device.StopScanning()
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initilize device: %v", err)
	}

	select {
	case r := <-done:
		return r.Control, r.error
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timed out finding device")
	}
}

func (c *Control) Move(m Move) {
	c.cmds <- m
}

func (c *Control) setBitsnap(n byte, value byte) {
	c.writeBLECmd(0x0A, []byte{n, value})
}

func (c *Control) motor1(value byte) {
	c.setBitsnap(0, value)
}

func (c *Control) motor2(value byte) {
	c.setBitsnap(2, value)
}

func (c *Control) servo(value byte) {
	c.setBitsnap(1, value)
}

func (c *Control) reset() {
	c.writeBLECmd(0x0E, []byte{0X89, 0X7F, 0X89})
}

func (c *Control) red(value byte) {
	c.led(value, 0, 0)
}

func (c *Control) green(value byte) {
	c.led(0, value, 0)
}

func (c *Control) blue(value byte) {
	c.led(0, 0, value)
}

func (c *Control) led(r byte, g byte, b byte) {
	c.writeBLECmd(0x09, []byte{r, g, b})
}

func (c *Control) writeBLECmd(cmd byte, data []byte) error {
	return c.peripheral.WriteCharacteristic(c.characteristic, bleCmd(cmd, data), true)
}

func findBitsnapControl(p gatt.Peripheral) *gatt.Characteristic {
	service, _ := gatt.ParseUUID(robotService)
	ss, _ := p.DiscoverServices([]gatt.UUID{service})
	for _, s := range ss {
		cs, _ := p.DiscoverCharacteristics([]gatt.UUID{service}, s)
		for _, c := range cs {
			ds, _ := p.DiscoverDescriptors(nil, c)
			for _, d := range ds {
				b, _ := p.ReadDescriptor(d)
				if string(b) == robotBitsnap {
					return c
				}
			}
		}
	}
	log.Fatalf("Unable to find Bitsnap Control characteristic")
	return nil
}

func bleCmd(cmd byte, data []byte) []byte {
	var b bytes.Buffer
	b.WriteByte(cmd << 1)
	b.WriteByte(byte(len(data)))
	b.Write(data)
	binary.Write(&b, binary.BigEndian, crc16.ChecksumCCITTFalse(data))
	return b.Bytes()
}
