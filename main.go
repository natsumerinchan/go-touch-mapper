package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/kenshaw/evdev"
)

func create_event_reader(indexes []int, running *bool) chan *event_pack {
	reader := func(event_reader chan *event_pack, index int, running *bool) {
		fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
		if err != nil {
			log.Fatal(err)
		}
		d := evdev.Open(fd)
		defer d.Close()
		ch := d.Poll(context.Background())
		events := make([]*evdev.Event, 0)
		dev_name := d.Name()
		fmt.Println("getevent from ", dev_name)
		d.Lock()
		for *running {
			event := <-ch
			if event.Type == evdev.SyncReport {
				pack := &event_pack{
					dev_name: dev_name,
					events:   events,
				}
				event_reader <- pack
				events = make([]*evdev.Event, 0)

			} else {
				events = append(events, &event.Event)
			}
		}
	}
	event_reader := make(chan *event_pack)
	for _, index := range indexes {
		go reader(event_reader, index, running)
	}
	return event_reader
}

func handel_touch(control_ch chan *touch_control_pack) {
	for {
		control_data := <-control_ch
		fmt.Printf("%+v\n", control_data)
	}
}

func handel_u_input(u_input chan *u_input_control_pack) {
	for {
		event_pack := <-u_input
		fmt.Println(event_pack)
	}
}

func main() {

	running := true

	event_reader := create_event_reader([]int{15, 17}, &running)

	touch_controller := make(chan *touch_control_pack)

	u_input := make(chan *u_input_control_pack)

	go handel_u_input(u_input)
	handel_u_input_interface(u_input)

	// return

	go direct_handel_touch(touch_controller)
	//注意  touch事件传递的XY坐标时为了直接写入触屏event的
	//并且只能在横屏模式下使用
	//而触屏event不会因为屏幕方向而改变坐标系
	//但是inputManager会
	//且程序时运行在横屏模式下的 即原本坐标就经过一次转换了
	//所以在直接写event无需转换而inputManager需要

	touchHandler := NewTouchHandler("EXAMPLE.JSON", event_reader, touch_controller, u_input)
	go touchHandler.auto_handel_view_release()
	go touchHandler.loop_handel_wasd_wheel()
	go touchHandler.loop_handel_rs_move()
	go touchHandler.handel_event()

	// th := TouchHandler{
	// 	id: 0,
	// }
	// fmt.Printf("%+v\n", th)

	for {
	}
	touchHandler.stop()
	// for {
	// 	select {
	// 	case control_data := <-touch_controller:
	// 		fmt.Println(control_data.action, control_data.id, control_data.x, control_data.y)
	// 	case event_pack := <-u_input:
	// 		fmt.Println(event_pack.dev_name)
	// 	}
	// }
}
