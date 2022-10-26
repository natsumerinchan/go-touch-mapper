package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
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

type touch_control_func func(data touch_control_pack)

func create_event_reader(indexes map[int]bool) chan *event_pack {
	reader := func(event_reader chan *event_pack, index int) {
		fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
		if err != nil {
			logger.Errorf("create_event_reader error:%v", err)
			return
		}
		d := evdev.Open(fd)
		defer d.Close()
		event_ch := d.Poll(context.Background())
		events := make([]*evdev.Event, 0)
		dev_name := d.Name()
		logger.Infof("开始读取设备 : %s", dev_name)
		d.Lock()
		defer d.Unlock()
		for {
			select {
			case <-global_close_signal:
				logger.Infof("释放设备 : %s  ", dev_name)
				return
			case event := <-event_ch:
				if event == nil {
					logger.Warnf("null event from %s , reader stopped !", dev_name)
					return
				} else if event.Type == evdev.SyncReport {
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
	for index, _ := range indexes {
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
		logger.Errorf("udp error %v", err)
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
	logger.Infof("已准备接收远程事件: 0.0.0.0:%d", port)
	for {
		select {
		case <-global_close_signal:
			return
		case pack := <-recv_ch:
			// logger.Debugf("%v", pack)
			event_count := int(pack[0])
			events := make([]*evdev.Event, 0)
			for i := 0; i < event_count; i++ {
				event := &evdev.Event{
					Type:  evdev.EventType(uint16(binary.LittleEndian.Uint16(pack[8*i+1 : 8*i+3]))),
					Code:  uint16(binary.LittleEndian.Uint16(pack[8*i+3 : 8*i+5])),
					Value: int32(binary.LittleEndian.Uint32(pack[8*i+5 : 8*i+9])),
				}
				// logger.Debugf("%v", event)
				events = append(events, event)
			}
			e_pack := &event_pack{
				dev_name: string(pack[event_count*8+1:]),
				events:   events,
			}
			ch <- e_pack
			// logger.Debugf("接收到事件 : %v", e_pack)
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

type dev_type uint8

const (
	type_mouse    = dev_type(0)
	type_keyboard = dev_type(1)
	type_joystick = dev_type(2)
	type_touch    = dev_type(3)
	type_unknown  = dev_type(4)
)

func check_dev_type(dev *evdev.Evdev) dev_type {
	abs := dev.AbsoluteTypes()
	key := dev.KeyTypes()
	rel := dev.RelativeTypes()
	_, MTPositionX := abs[evdev.AbsoluteMTPositionX]
	_, MTPositionY := abs[evdev.AbsoluteMTPositionY]
	_, MTSlot := abs[evdev.AbsoluteMTSlot]
	_, MTTrackingID := abs[evdev.AbsoluteMTTrackingID]
	if MTPositionX && MTPositionY && MTSlot && MTTrackingID {
		return type_touch //触屏检测这几个abs类型即可
	}
	_, RelX := rel[evdev.RelativeX]
	_, RelY := rel[evdev.RelativeY]
	_, HWheel := rel[evdev.RelativeHWheel]
	_, MouseLeft := key[evdev.BtnLeft]
	_, MouseRight := key[evdev.BtnRight]
	_, MouseMiddle := key[evdev.BtnMiddle]
	if RelX && RelY && HWheel && MouseLeft && MouseRight && MouseMiddle {
		return type_mouse //鼠标 检测XY 滚轮 左右中键
	}
	keyboard_keys := true
	for i := evdev.KeyEscape; i <= evdev.KeyScrollLock; i++ {
		_, ok := key[i]
		keyboard_keys = keyboard_keys && ok
	}
	if keyboard_keys {
		return type_keyboard //键盘 检测keycode(1-70)
	}

	axis_count := 0
	for i := evdev.AbsoluteX; i <= evdev.AbsoluteRZ; i++ {
		_, ok := abs[i]
		if ok {
			axis_count++
		}
	}
	LS_RS := axis_count >= 4

	key_count := 0
	for i := evdev.BtnA; i <= evdev.BtnZ; i++ {
		_, ok := key[i]
		if ok {
			key_count++
		}
	}
	A_B_X_Y := key_count >= 4

	if LS_RS && A_B_X_Y {
		return type_joystick //手柄 检测LS,RS A,B,X,Y
	}
	return type_unknown
}

func get_possible_device_indexes() map[int]dev_type {
	files, _ := ioutil.ReadDir("/dev/input")
	result := make(map[int]dev_type)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if len(file.Name()) <= 5 {
			continue
		}
		if file.Name()[:5] != "event" {
			continue
		}
		index, _ := strconv.Atoi(file.Name()[5:])
		fd, err := os.OpenFile(fmt.Sprintf("/dev/input/%s", file.Name()), os.O_RDONLY, 0)
		if err != nil {
			logger.Errorf("open dev error:%v", err)
		}
		d := evdev.Open(fd)
		defer d.Close()
		devType := check_dev_type(d)
		if devType != type_unknown {
			result[index] = devType
		}
	}
	return result
}

func get_dev_name_by_index(index int) string {
	fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
	if err != nil {
		return "read name error"
	}
	d := evdev.Open(fd)
	defer d.Close()
	return d.Name()
}

// func get_dev_phys_info_by_index(index int) string {
// 	fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
// 	if err != nil {
// 		return "read name error"
// 	}
// 	d := evdev.Open(fd)
// 	defer d.Close()
// 	return d.Name()
// 	// return d
// }

func execute_view_move(handelerInstance *TouchHandler, x, stepValue, sleepMS int) {
	handelerInstance.handel_view_move(0, 0)
	time.Sleep(time.Millisecond * time.Duration(sleepMS))
	if x > 0 {
		steps := x / stepValue
		for i := 0; i < steps; i++ {
			if handelerInstance.map_on == false {
				break
			}
			handelerInstance.handel_view_move(int32(stepValue), 0)
			time.Sleep(time.Millisecond * time.Duration(sleepMS))
		}
		handelerInstance.handel_view_move(int32(x%stepValue), 0)
	} else {
		steps := -x / stepValue
		for i := 0; i < steps; i++ {
			if handelerInstance.map_on == false {
				break
			}
			handelerInstance.handel_view_move(-int32(stepValue), 0)
			time.Sleep(time.Millisecond * time.Duration(sleepMS))
		}
		handelerInstance.handel_view_move(-int32(x%stepValue), 0)
	}
}

func stdin_control_view_move(handelerInstance *TouchHandler) {
	logger.Info("输入数值以精确控制view移动 [ x | x 步长 间隔 ]")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		args := strings.Split(scanner.Text(), " ")
		var x, stepValue, sleepMS int
		var err error
		if len(args) == 1 {
			x, err = strconv.Atoi(args[0])
			if err != nil {
				logger.Errorf("输入错误: %s", err)
				continue
			} else {
				stepValue = 24
				sleepMS = 16
			}

		} else if len(args) == 3 {
			x, err = strconv.Atoi(args[0])
			if err != nil {
				logger.Errorf("输入错误: %s", err)
				continue
			}
			stepValue, err = strconv.Atoi(args[1])
			if err != nil {
				logger.Errorf("输入错误: %s", err)
				continue
			}
			sleepMS, err = strconv.Atoi(args[2])
			if err != nil {
				logger.Errorf("输入错误: %s", err)
				continue
			}
		} else {
			logger.Error("参数错误 usage:x | x stepValue sleepMS")
			continue
		}
		logger.Infof("x: %d stepValue: %d sleepMS: %d", x, stepValue, sleepMS)
		if handelerInstance.map_on {
			execute_view_move(handelerInstance, x, stepValue, sleepMS)
		} else {
			logger.Info("等待映射开关打开中...")
			for {
				if handelerInstance.map_on == false {
					time.Sleep(time.Duration(100) * time.Millisecond)
				} else {
					execute_view_move(handelerInstance, x, stepValue, sleepMS)
					break
				}
			}
		}
	}
}

func get_MT_size(indexes map[int]bool) (int32, int32) { //获取MTPositionX和MTPositionY的max值
	//在1+7p上是等于get_wm_size
	//但是红魔7sp应该是为了更精确，使用了缩放,实际的数值为 rawValue >> 3
	// rawValue * wm_size / mt_size = true_value
	for index, _ := range indexes {
		fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
		if err != nil {
			logger.Errorf("get_MT_size error:%v", err)
		}
		d := evdev.Open(fd)
		defer d.Close()
		abs := d.AbsoluteTypes()
		MTPositionX, _ := abs[evdev.AbsoluteMTPositionX]
		MTPositionY, _ := abs[evdev.AbsoluteMTPositionY]
		return MTPositionX.Max, MTPositionY.Max
	}
	return int32(1), int32(1)
}

func test() {
	for i := 0; i < 16; i++ {
		start := time.Now()
		end := time.Now()
		cost := end.Nanosecond() - start.Nanosecond()
		fmt.Printf("cost : %v ns\n", cost)
	}

}

func chan_test() {
	ch := make(chan time.Time)
	go func() {
		for i := 0; i < 16; i++ {
			start := <-ch
			end := time.Now()
			cost := end.Nanosecond() - start.Nanosecond()
			fmt.Printf("cost : %v ns\n", cost)
		}
	}()
	for i := 0; i < 16; i++ {
		time.Sleep(10 * time.Duration(time.Millisecond))
		ch <- time.Now()
	}
}

func main() {
	// test()
	// fmt.Printf("====================")
	// chan_test()
	// return
	parser := argparse.NewParser("go-touch-mapper", " ")
	var auto_detect *bool = parser.Flag("a", "auto-detect", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "自动检测设备",
	})

	var create_js_info *bool = parser.Flag("", "create-js-info", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "创建手柄配置文件",
	})

	var eventList *[]int = parser.IntList("e", "event", &argparse.Options{
		Required: false,
		Help:     "键盘或鼠标或手柄的设备号",
	})
	var mixTouchDisabled *bool = parser.Flag("t", "touch-disabled", &argparse.Options{
		Required: false,
		Help:     "关闭触屏混合",
		Default:  false,
	})
	var configPath *string = parser.String("c", "config", &argparse.Options{
		Required: false,
		Help:     "配置文件路径",
	})

	var usingInputManager *bool = parser.Flag("i", "inputManager", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "使用inputManager控制触摸",
	})

	var using_remote_control *bool = parser.Flag("r", "remoteControl", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "是否从UDP接收远程事件",
	})

	var udp_port *int = parser.Int("p", "port", &argparse.Options{
		Required: false,
		Help:     "指定监听远程事件的UDP端口号",
		Default:  61069,
	})

	var using_v_mouse *bool = parser.Flag("v", "v-mouse", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "用触摸操作模拟鼠标,需要额光标外显示程序",
	})

	var view_release_timeout *int = parser.Int("", "auto-release", &argparse.Options{
		Required: false,
		Help:     "触发视角自动释放所需的静止ms数,50ms为检查单位,置0禁用",
		Default:  200,
	})

	var measure_sensitivity_mode *bool = parser.Flag("", "measure-mode", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "显示视角移动像素计数,且可输入数值模拟滑动,方向键可微调",
	})

	var debug_mode *bool = parser.Flag("d", "debug_mode", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "打印debug信息",
	})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}
	if *debug_mode {
		logger.WithDebug()
		logger.Debug("debug on")
	}

	auto_detect_result := make(map[int]dev_type)
	if *auto_detect {
		auto_detect_result = get_possible_device_indexes()
		devTypeFriendlyName := map[dev_type]string{
			type_mouse:    "鼠标",
			type_keyboard: "键盘",
			type_joystick: "手柄",
			type_touch:    "触屏",
			type_unknown:  "未知",
		}
		for index, devType := range auto_detect_result {
			devName := get_dev_name_by_index(index)
			logger.Infof("检测到设备 %s(/dev/input/event%d) : %s", devName, index, devTypeFriendlyName[devType])
		}
	}
	if *create_js_info {
		if *auto_detect {
			js_events := make([]int, 0)
			for index, devType := range auto_detect_result {
				if devType == type_joystick {
					js_events = append(js_events, index)
				}
			}
			if len(js_events) == 1 {
				create_js_info_file(js_events[0])
			} else {
				if len(js_events) == 0 {
					logger.Warn("未自动检测到手柄,请手动指定")
				} else {
					logger.Warn("检测到多个手柄,无法使用 -a 自动检测,请使用 -e 手动指定")
				}
			}
		} else {
			if len(*eventList) == 0 {
				logger.Warn("请至少指定一个设备号")
			} else if len(*eventList) > 1 {
				logger.Warn("最多只能指定一个设备号")
			} else {
				create_js_info_file((*eventList)[0])
			}
		}
	} else {
		eventSet := make(map[int]bool)
		for _, index := range *eventList {
			eventSet[index] = true
		}
		for k, v := range auto_detect_result {
			if v == type_joystick || v == type_keyboard || v == type_mouse {
				eventSet[k] = true
			}
		}
		events_ch := create_event_reader(eventSet)

		u_input_control_ch := make(chan *u_input_control_pack)
		fileted_u_input_control_ch := make(chan *u_input_control_pack)
		touch_event_ch := make(chan *event_pack)
		max_mt_x, max_mt_y := int32(1), int32(1)
		if !*mixTouchDisabled {
			for k, v := range auto_detect_result {
				if v == type_touch {
					logger.Infof("启用触屏混合 : event%d", k)
					touch_event_ch = create_event_reader(map[int]bool{k: true})
					max_mt_x, max_mt_y = get_MT_size(map[int]bool{k: true})
					break
				}
			}
		}

		go listen_device_orientation()

		go handel_u_input_mouse_keyboard(fileted_u_input_control_ch)

		var couch_control_func touch_control_func
		if *usingInputManager {
			logger.Info("触屏控制将使用inputManager处理")
			couch_control_func = handel_touch_using_input_manager()
		} else {
			couch_control_func = handel_touch_using_vTouch()
		}

		map_switch_signal := make(chan bool) //通知虚拟鼠标当前为鼠标还是映射模式
		touchHandler := InitTouchHandler(
			*configPath,
			events_ch,
			couch_control_func,
			u_input_control_ch,
			!*usingInputManager,
			map_switch_signal,
			*measure_sensitivity_mode,
		)

		go touchHandler.mix_touch(touch_event_ch, max_mt_x, max_mt_y)
		go touchHandler.auto_handel_view_release(*view_release_timeout)
		go touchHandler.loop_handel_wasd_wheel()
		go touchHandler.loop_handel_rs_move()
		go touchHandler.handel_event()
		if *using_v_mouse {
			v_mouse := init_v_mouse_controller(touchHandler, u_input_control_ch, fileted_u_input_control_ch, map_switch_signal)
			go v_mouse.main_loop()
		} else {
			go (func() {
				for {
					select {
					case tmp := <-u_input_control_ch:
						fileted_u_input_control_ch <- tmp
					case <-map_switch_signal:
					}
				}
			})()
		}

		if *using_remote_control {
			go udp_event_injector(events_ch, *udp_port)
		}

		if *measure_sensitivity_mode {
			go stdin_control_view_move(touchHandler)
		}

		exitChan := make(chan os.Signal)
		signal.Notify(exitChan, os.Interrupt, os.Kill, syscall.SIGTERM)
		<-exitChan
		close(global_close_signal)
		logger.Info("已停止")
		time.Sleep(time.Millisecond * 40)
	}
}
