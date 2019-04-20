package temperusb

import (
	"fmt"
	"time"

	"github.com/google/gousb"
	log "github.com/sirupsen/logrus"
)

const (
	INTERFACE1 uint16 = 0x00
	INTERFACE2 uint16 = 0x01

	reqIntLen   int    = 8
	InEndpoint  uint16 = 0x82
	OutEndpoint uint16 = 0x00
	tempOffset  int    = 2
)

var (
	Timeout = 5 * time.Second
	uTemp   = []byte{0x01, 0x80, 0x33, 0x01, 0x00, 0x00, 0x00, 0x00}
	uIni1   = []byte{0x01, 0x82, 0x77, 0x01, 0x00, 0x00, 0x00, 0x00}
	uIni2   = []byte{0x01, 0x86, 0xff, 0x01, 0x00, 0x00, 0x00, 0x00}
)

type Temper struct {
	ctx   *gousb.Context
	dev   *gousb.Device
	iface *gousb.Interface
}

func New() (*Temper, error) {
	ctx := gousb.NewContext()
	ctx.Debug(0)

	// OpenDevices is used to find the devices to open.
	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		// After inspecting the descriptor, return true or false depending on whether
		// the device is "interesting" or not.  Any descriptor for which true is returned
		// opens a Device which is retuned in a slice (and must be subsequently closed).
		if desc.Vendor == gousb.ID(0x0c45) && desc.Product == gousb.ID(0x7401) {
			log.Debugf("Found device: %v", desc)
			return true
		}
		return false
	})

	// All Devices returned from OpenDevices must be closed.

	// OpenDevices can occaionally fail, so be sure to check its return value.
	if err != nil {
		return nil, fmt.Errorf("Failed listing devices: %v", err)
	}

	// Must have found at least one device
	if len(devs) == 0 {
		return nil, fmt.Errorf("Found no TEMPER devices")
	} else {
		log.Infof("Found %d devices", len(devs))
	}

	//FIXME: Currently, we just do the first device

	// Close any others that were opened
	/*
		defer func() {
			for _, dev := range devs[1:] {
				dev.Close()
			}
		}()
	*/

	t := &Temper{}
	t.ctx = ctx
	t.dev = devs[0]

	// Reset
	log.Debugf("Reset device")
	t.dev.Reset()

	//t.dev.ControlTimeout = Timeout

	// Detach INTERFACE1 & INTERFACE2
	log.Debugf("Set auto-detach")
	if err := t.dev.SetAutoDetach(true); err != nil {
		t.dev.Close()
		return nil, fmt.Errorf("Failed to set auto-detach: %v", err)
	}

	// Set configuration to 0x01
	log.Debugf("Set configuration to 0x01")
	config, err := t.dev.Config(0x01)
	if err != nil {
		t.dev.Close()
		return nil, fmt.Errorf("Failed to configure device: %v", err)
	}

	// Claim interfaces
	for i := 0; i < 2; i++ {
		log.Debugf("Claim iface-%v", i)
		iface, err := config.Interface(i, 0)
		if err != nil {
			t.dev.Close()
			return nil, fmt.Errorf("Failed to claim iface-%v: %v", i, err)
		}
		t.iface = iface
	}

	// ini_control_transfer
	log.Debugf("ini_control_transfer")
	if ret, err := t.dev.Control(0x21, 0x09, 0x0201, 0x00, []byte{0x01, 0x01}); err != nil {
		t.dev.Close()
		return nil, fmt.Errorf("Failed to run ini_control_transfer: %v (error code: %v)", err, ret)
	}

	// Control transfers
	dummyBytes := make([]byte, reqIntLen)

	log.Debugf("control transfer uTemp")
	if ret, err := t.controlTransfer(uTemp); err != nil {
		t.dev.Close()
		return nil, fmt.Errorf("Failed to run control transfer uTemp: %v (error code: %v)", err, ret)
	}
	log.Debugf("interrupt_read uTemp")
	t.interrupt_read(dummyBytes)
	//TODO: Interrupt read

	log.Debugf("control transfer uIni1")
	if ret, err := t.controlTransfer(uIni1); err != nil {
		t.dev.Close()
		return nil, fmt.Errorf("Failed to run control transfer uIni1: %v (error code: %v)", err, ret)
	}
	//TODO: Interrupt read
	log.Debugf("interrupt_read uIni1")
	t.interrupt_read(dummyBytes)

	log.Debugf("control transfer uIni2")
	if ret, err := t.controlTransfer(uIni2); err != nil {
		t.dev.Close()
		return nil, fmt.Errorf("Failed to run control transfer uIni2: %v (error code: %v)", err, ret)
	}
	//TODO: Interrupt read
	log.Debugf("interrupt_read uIni2-1")
	t.interrupt_read(dummyBytes)
	//TODO: Interrupt read
	log.Debugf("interrupt_read uIni2-2")
	t.interrupt_read(dummyBytes)

	return t, nil
}

func (t *Temper) GetTemperature() (float64, error) {
	log.Debugf("temperature() -> controlTransfer")
	if ret, err := t.controlTransfer(uTemp); err != nil {
		return 0.0, fmt.Errorf("Failed to read temperature: %v (error code: %v)", err, ret)
	}
	// TODO: interrupt_read_temperature
	data := make([]byte, reqIntLen)
	if ret, err := t.interrupt_read(data); err != nil {
		return 0.0, fmt.Errorf("Failed to execute interrupt_read: %v (error code: %v)", err, ret)
	}
	if len(data) != reqIntLen {
		return 0.0, fmt.Errorf("Failed USB interrupt read3: %v", len(data))
	}
	h := uint8(data[tempOffset])
	l := uint8(data[tempOffset+1])
	temp := float64(h) + float64(l)/256.0
	return temp, nil
}

func (t *Temper) controlTransfer(bytes []byte) (int, error) {
	return t.dev.Control(0x21, 0x09, 0x0200, 0x01, bytes)
}

func (t *Temper) interrupt_read(data []byte) (int, error) {
	iface := t.iface
	ep, err := iface.InEndpoint(0x82)
	if err != nil {
		return -1, err
	}
	return ep.Read(data)
}
