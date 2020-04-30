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

type command struct {
	cmd byte
	data []byte
}

type Control struct {
	cmds           chan command
	peripheral     gatt.Peripheral
	characteristic *gatt.Characteristic
}

func (c *Control) SetBitsnap(n byte, value byte) {
	c.cmds <- command{
		cmd: 0x0A,
		data: []byte{n, value},
	}
}

func (c *Control) Motor1(value byte) {
	c.SetBitsnap(0, value)
}

func (c *Control) Motor2(value byte) {
	c.SetBitsnap(2, value)
}

func (c *Control) Servo(value byte) {
	c.SetBitsnap(1, value)
}

func (c *Control) Reset() {
	c.cmds <- command{
		cmd: 0x0E,
		data: []byte{0X7F, 0X7F, 0X7F},
	}
}

func (c *Control) Red(value byte) {
	c.LED(value, 0, 0)
}

func (c *Control) Green(value byte) {
	c.LED(0, value, 0)
}

func (c *Control) Blue(value byte) {
	c.LED(0, 0, value)
}

func (c *Control) LED(r byte, g byte, b byte) {
	c.cmds <- command{
		cmd: 0x09,
		data: []byte{r, g, b},
	}
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
				cmds:           make(chan command, 100),
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

			ctrl.Reset()

			for c := range ctrl.cmds {
				err = ctrl.writeBLECmd(c.cmd, c.data)
				if err != nil {
					log.Printf("error processing command: %v", err)
				}
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
