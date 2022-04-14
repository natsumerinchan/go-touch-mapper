//用于直接写dev来控制触屏的接口

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unsafe"

	"github.com/kenshaw/evdev"
)

const (
	ABS_MT_POSITION_X  = 0x35
	ABS_MT_POSITION_Y  = 0x36
	ABS_MT_SLOT        = 0x2F
	ABS_MT_TRACKING_ID = 0x39
	EV_SYN             = 0x00
	EV_KEY             = 0x01
	EV_REL             = 0x02
	EV_ABS             = 0x03
	REL_X              = 0x00
	REL_Y              = 0x01
	REL_WHEEL          = 0x08
	REL_HWHEEL         = 0x06
	SYN_REPORT         = 0x00
	BTN_TOUCH          = 0x14A
)

var sizeofEvent int

func sendEvents(fd *os.File, events []*evdev.Event) {
	buf := make([]byte, sizeofEvent*len(events))
	for i, event := range events {
		copy(buf[i*sizeofEvent:], (*(*[1<<27 - 1]byte)(unsafe.Pointer(event)))[:sizeofEvent])
	}
	// fmt.Printf("bytes %+v\n", buf)
	n, err := fd.Write(buf)
	if err != nil {
		fmt.Println(err, n)
	}
}

func get_orientation() uint8 {
	cmd := "dumpsys input | grep 'SurfaceOrientation' | awk '{{print $2}}'"
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return 0
	}
	out_str := string(out)
	replaced := strings.Replace(out_str, "\n", "", -1)
	uin_res, _ := strconv.ParseUint(replaced, 10, 8)
	fmt.Printf("%s %+v\n", out_str, uin_res)
	return uint8(uin_res)
}

func transform_orientation(control_data *touch_control_pack, orientation uint8) (int32, int32) {
	switch orientation {
	case 0:
		return control_data.x, control_data.y
	case 1:
		return control_data.screen_y - control_data.y, control_data.x
	case 3:
		return control_data.y, control_data.screen_x - control_data.x
	default:
		return control_data.x, control_data.y
	}
}

func direct_handel_touch(control_ch chan *touch_control_pack) {
	sizeofEvent = int(unsafe.Sizeof(evdev.Event{}))
	ev_sync := evdev.Event{Type: EV_SYN, Code: 0, Value: 0}
	init_orientation := get_orientation()
	fmt.Printf("init_orientation %d\n", init_orientation)
	var count int32 = 0    //BTN_TOUCH 申请时为1 则按下 释放时为0 则松开
	var last_id int32 = -1 //ABS_MT_SLOT last_id每次动作后修改 如果不等则额外发送MT_SLOT事件
	fd, err := os.OpenFile("/dev/input/event5", os.O_RDWR, 0)
	if err != nil {
		fmt.Println(err, fd)
	}
	for {
		control_data := <-control_ch
		x, y := transform_orientation(control_data, init_orientation)

		write_events := make([]*evdev.Event, 0)
		if control_data.action == TouchActionRequire {
			last_id = control_data.id
			write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_SLOT, Value: control_data.id})
			write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_TRACKING_ID, Value: control_data.id})
			count += 1
			if count == 1 {
				write_events = append(write_events, &evdev.Event{Type: EV_KEY, Code: BTN_TOUCH, Value: DOWN})
			}
			write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_X, Value: x})
			write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_Y, Value: y})
			write_events = append(write_events, &ev_sync)
			sendEvents(fd, write_events)
		} else if control_data.action == TouchActionRelease {
			if last_id != control_data.id {
				last_id = control_data.id
				write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_SLOT, Value: control_data.id})
			}
			write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_TRACKING_ID, Value: -1})
			count -= 1
			if count == 0 {
				write_events = append(write_events, &evdev.Event{Type: EV_KEY, Code: BTN_TOUCH, Value: UP})
			}
			write_events = append(write_events, &ev_sync)
			sendEvents(fd, write_events)
		} else if control_data.action == TouchActionMove {
			if last_id != control_data.id {
				last_id = control_data.id
				write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_SLOT, Value: control_data.id})
			}
			write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_X, Value: x})
			write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_Y, Value: y})
			write_events = append(write_events, &ev_sync)
			sendEvents(fd, write_events)
		}
	}
}
