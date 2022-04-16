package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/akamensky/argparse"
	"github.com/kenshaw/evdev"
)

type event_pack struct {
	//表示一个动作 由一系列event组成
	dev_name string
	events   []*evdev.Event
}

type touch_control_pack struct {
	//触屏控制信息
	action   int8
	id       int32
	x        int32
	y        int32
	screen_x int32
	screen_y int32
}

type u_input_control_pack struct {
	action int8
	arg1   int32
	arg2   int32
}

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
		// fmt.Println("", dev_name)
		fmt.Printf("开始读取设备 : %s\n", dev_name)
		d.Lock()
		defer d.Unlock()
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

func main() {

	parser := argparse.NewParser("go-touch-mappeer", " ")

	var eventList *[]int = parser.IntList("e", "event", &argparse.Options{
		Required: false,
		Help:     "键盘或鼠标或手柄的设备号",
	})
	var touchIndex *int = parser.Int("t", "touch", &argparse.Options{
		Required: false,
		Help:     "触屏设备号,可选,当指定时可同时使用映射与触屏而不冲突",
		Default:  -1,
	})
	var configPath *string = parser.String("c", "config", &argparse.Options{
		Required: true,
		Help:     "配置文件路径",
	})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
	}

	running := true
	event_reader := create_event_reader(*eventList, &running)

	var touch_channel chan *event_pack
	if *touchIndex != -1 {
		touch_channel = create_event_reader([]int{*touchIndex}, &running)
	}

	touch_controller := make(chan *touch_control_pack)

	u_input := make(chan *u_input_control_pack)

	go handel_u_input_mouse_keyboard(u_input)
	go handel_touch_using_vTouch(touch_controller)

	touchHandler := NewTouchHandler(*configPath, event_reader, touch_controller, u_input)

	if *touchIndex != -1 {
		fmt.Printf("启用触屏混合 : event%d\n", *touchIndex)
		go touchHandler.mix_touch(touch_channel)
	}

	go touchHandler.auto_handel_view_release()
	go touchHandler.loop_handel_wasd_wheel()
	go touchHandler.loop_handel_rs_move()
	go touchHandler.handel_event()

	for {
	}
	// touchHandler.stop()

}
