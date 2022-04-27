package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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

func create_event_reader(indexes []int) chan *event_pack {
	reader := func(event_reader chan *event_pack, index int) {
		fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
		if err != nil {
			log.Fatal(err)
		}
		d := evdev.Open(fd)
		defer d.Close()
		event_ch := d.Poll(context.Background())
		events := make([]*evdev.Event, 0)
		dev_name := d.Name()
		fmt.Printf("开始读取设备 : %s\n", dev_name)
		d.Lock()
		defer d.Unlock()
		for {
			select {
			case <-global_close_signal:
				fmt.Printf("释放设备 : %s \n ", dev_name)
				return
			case event := <-event_ch:
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

	}
	event_reader := make(chan *event_pack)
	for _, index := range indexes {
		go reader(event_reader, index)
	}
	return event_reader
}

func udp_event_injector(ch chan *event_pack, port int) {
	listen, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: port,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	defer listen.Close()

	recv_ch := make(chan []byte)
	go func() {
		for {
			var buf [1024]byte
			n, _, err := listen.ReadFromUDP(buf[:])
			if err != nil {
				break
			}
			recv_ch <- buf[:n]
		}
	}()
	fmt.Printf("已准备接收远程事件: 0.0.0.0:%d\n", port)
	for {
		select {
		case <-global_close_signal:
			return
		case pack := <-recv_ch:
			// fmt.Printf("%v\n", pack)
			event_count := int(pack[0])
			events := make([]*evdev.Event, 0)
			for i := 0; i < event_count; i++ {
				event := &evdev.Event{
					Type:  evdev.EventType(uint16(binary.LittleEndian.Uint16(pack[8*i+1 : 8*i+3]))),
					Code:  uint16(binary.LittleEndian.Uint16(pack[8*i+3 : 8*i+5])),
					Value: int32(binary.LittleEndian.Uint32(pack[8*i+5 : 8*i+9])),
				}
				// fmt.Printf("%v\n", event)
				events = append(events, event)
			}
			e_pack := &event_pack{
				dev_name: string(pack[event_count*8+1:]),
				events:   events,
			}
			ch <- e_pack
			// fmt.Printf("接收到事件 : %v\n", e_pack)
		}
	}
}

var global_close_signal = make(chan bool) //仅会在程序退出时关闭  不用于其他用途
var global_device_orientation int32 = 0

func get_device_orientation() int32 {
	cmd := "dumpsys input | grep -i surfaceorientation | awk '{ print $2 }'"
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return 0
	} else {
		result, err := strconv.Atoi(string(out[0:1]))
		if err != nil {
			return 0
		} else {
			return int32(result)
		}
	}
}

func listen_device_orientation() {
	for {
		select {
		case <-global_close_signal:
			return
		default:
			global_device_orientation = get_device_orientation()
			time.Sleep(time.Duration(1) * time.Second)
		}
	}
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

	var usingInputManager *bool = parser.Flag("i", "inputManager", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "是否使用inputManager,需开启额外控制进程",
	})

	var using_remote_control *bool = parser.Flag("r", "remoteControl", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "是否从UDP接收远程事件",
	})

	var udp_port *int = parser.Int("p", "port", &argparse.Options{
		Required: false,
		Help:     "指定监听远程事件的UDP端口号,默认61069",
		Default:  61069,
	})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	events_ch := create_event_reader(*eventList)
	touch_control_ch := make(chan *touch_control_pack)
	u_input_control_ch := make(chan *u_input_control_pack)
	touch_event_ch := make(chan *event_pack)

	// *touchIndex = -1 // 暂时取消触屏混合的支持 在坐标系转换结束后再重新设计
	if *touchIndex != -1 {
		fmt.Printf("启用触屏混合 : event%d\n", *touchIndex)
		touch_event_ch = create_event_reader([]int{*touchIndex})
	}

	go listen_device_orientation()

	go handel_u_input_mouse_keyboard(u_input_control_ch)
	if *usingInputManager {
		fmt.Println("触屏控制将使用inputManager处理")
		go handel_touch_using_input_manager(touch_control_ch) //先统一坐标系
	} else {
		go handel_touch_using_vTouch(touch_control_ch) //然后再处理转换旋转后的坐标
	}

	touchHandler := InitTouchHandler(*configPath, events_ch, touch_control_ch, u_input_control_ch, !*usingInputManager)
	go touchHandler.mix_touch(touch_event_ch)
	go touchHandler.auto_handel_view_release()
	go touchHandler.loop_handel_wasd_wheel()
	go touchHandler.loop_handel_rs_move()
	go touchHandler.handel_event()

	if *using_remote_control {
		go udp_event_injector(events_ch, *udp_port)
	}

	exitChan := make(chan os.Signal)
	signal.Notify(exitChan, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-exitChan
	close(global_close_signal)
	fmt.Println("已停止")
	time.Sleep(time.Millisecond * 40)
}
