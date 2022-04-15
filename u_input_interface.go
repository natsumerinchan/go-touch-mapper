package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"syscall"

	"github.com/lunixbochs/struc"
)

func toUInputName(name []byte) [uinputMaxNameSize]byte {
	var fixedSizeName [uinputMaxNameSize]byte
	copy(fixedSizeName[:], name)
	return fixedSizeName
}

func uInputDevToBytes(uiDev UinputUserDev) []byte {
	var buf bytes.Buffer
	_ = struc.PackWithOptions(&buf, &uiDev, &struc.Options{Order: binary.LittleEndian})
	return buf.Bytes()
}

func createDevice(f *os.File) (err error) {
	return ioctl(f.Fd(), UIDEVCREATE(), uintptr(0))
}

func create_u_input_touch_screen(width int, height int) *os.File {
	deviceFile, err := os.OpenFile("/dev/uinput", syscall.O_WRONLY|syscall.O_NONBLOCK, 0660)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	ioctl(deviceFile.Fd(), UISETEVBIT(), evKey)
	ioctl(deviceFile.Fd(), UISETKEYBIT(), 0x014a) //一个是BTN_TOUCH 一个不知道是啥
	ioctl(deviceFile.Fd(), UISETKEYBIT(), 0x003e) //是从手机直接copy出来的

	ioctl(deviceFile.Fd(), UISETEVBIT(), evAbs)
	ioctl(deviceFile.Fd(), UISETABSBIT(), absMtSlot)
	ioctl(deviceFile.Fd(), UISETABSBIT(), absMtTrackingId)

	ioctl(deviceFile.Fd(), UISETABSBIT(), absMtTouchMajor)
	ioctl(deviceFile.Fd(), UISETABSBIT(), absMtWidthMajor)
	ioctl(deviceFile.Fd(), UISETABSBIT(), absMtPositionX)
	ioctl(deviceFile.Fd(), UISETABSBIT(), absMtPositionY)

	ioctl(deviceFile.Fd(), UISETPROPBIT(), inputPropDirect)

	var absMin [absCnt]int32
	absMin[absMtPositionX] = 0
	absMin[absMtPositionY] = 0
	absMin[absMtTouchMajor] = 0
	absMin[absMtWidthMajor] = 0
	absMin[absMtSlot] = 0
	absMin[absMtTrackingId] = 0

	var absMax [absCnt]int32
	absMax[absMtPositionX] = int32(width)
	absMax[absMtPositionY] = int32(height)
	absMax[absMtTouchMajor] = 255
	absMax[absMtWidthMajor] = 0
	absMax[absMtSlot] = 255
	absMax[absMtTrackingId] = 65535

	uiDev := UinputUserDev{
		Name: toUInputName([]byte("v_touch_screen")),
		ID: InputID{
			BusType: 0,
			Vendor:  randUInt16Num(0x2000),
			Product: randUInt16Num(0x2000),
			Version: randUInt16Num(0x20),
		},
		EffectsMax: 0,
		AbsMax:     absMax,
		AbsMin:     absMin,
		AbsFuzz:    [absCnt]int32{},
		AbsFlat:    [absCnt]int32{},
	}
	deviceFile.Write(uInputDevToBytes(uiDev))
	createDevice(deviceFile)
	return deviceFile
}
