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

	event_reader := create_event_reader([]int{15, 16}, &running)

	touch_controller := make(chan *touch_control_pack)

	u_input := make(chan *u_input_control_pack)

	// go handel_touch(touch_controller)
	go direct_handel_touch(touch_controller)

	go handel_u_input(u_input)

	touchHandler := NewTouchHandler("EXAMPLE.JSON", event_reader, touch_controller, u_input)

	go touchHandler.auto_handel_view_release()
	go touchHandler.loop_handel_wasd_wheel()
	go touchHandler.loop_handel_rs_move()
	touchHandler.handel_event()

	// th := TouchHandler{
	// 	id: 0,
	// }
	// fmt.Printf("%+v\n", th)

	for {
	}
	// for {
	// 	select {
	// 	case control_data := <-touch_controller:
	// 		fmt.Println(control_data.action, control_data.id, control_data.x, control_data.y)
	// 	case event_pack := <-u_input:
	// 		fmt.Println(event_pack.dev_name)
	// 	}
	// }
}
