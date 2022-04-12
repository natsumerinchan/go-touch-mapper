package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/kenshaw/evdev"
)

type TouchHandler struct {
	events                  chan *event_pack
	touch_controller        chan *touch_control_pack
	u_input                 chan *u_input_control_pack
	map_on                  bool
	view_id                 int8
	stick_id                int8
	allocated_id            []bool
	config                  *simplejson.Json
	joystickInfo            map[string]*simplejson.Json
	screen_x                int
	screen_y                int
	screen_init_x           int
	screen_init_y           int
	screen_current_x        int
	screen_current_y        int
	view_lock               sync.Mutex
	auto_release_view_count int32
	abs_last                map[string]float64
}

const (
	TouchActionRequire int8 = 0
	TouchActionRelease int8 = 1
	TouchActionMove    int8 = 2
)

const (
	UInput_mouse_move int = 0
	UInput_mouse_btn  int = 1
	UInput_key_event  int = 2
)

const (
	DOWN int32 = 1
	UP   int32 = 0
)

var HAT_D_U map[string]([]int32) = map[string]([]int32){
	"0.5_1.0": []int32{1, DOWN},
	"0.5_0.0": []int32{0, DOWN},
	"1.0_0.5": []int32{1, UP},
	"0.0_0.5": []int32{0, UP},
}

var HAT0_KEYNAME map[string][]string = map[string][]string{
	"HAT0X": {"BTN_DPAD_LEFT", "BTN_DPAD_RIGHT"},
	"HAT0Y": {"BTN_DPAD_UP", "BTN_DPAD_DOWN"},
}

func NewTouchHandler(
	mapperFilePath string,
	events chan *event_pack,
	touch_controller chan *touch_control_pack,
	u_input chan *u_input_control_pack,
) *TouchHandler {
	content, _ := ioutil.ReadFile(mapperFilePath)
	config_json, _ := simplejson.NewJson(content)
	screen_x := config_json.Get("SCREEN").Get("SIZE").GetIndex(1).MustInt()
	screen_y := config_json.Get("SCREEN").Get("SIZE").GetIndex(0).MustInt()

	//对./目录下所有JSON文件
	//读取并以文件名作为key创建一个map
	files, _ := ioutil.ReadDir("./joystickInfos")
	joystickInfo := make(map[string]*simplejson.Json)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if file.Name()[len(file.Name())-5:] != ".json" {
			continue
		}
		content, _ := ioutil.ReadFile("./joystickInfos/" + file.Name())
		info, _ := simplejson.NewJson(content)
		joystickInfo[file.Name()[:len(file.Name())-5]] = info
	}
	fmt.Printf("joystickInfo:%v\n", joystickInfo)

	return &TouchHandler{
		events:                  events,
		touch_controller:        touch_controller,
		u_input:                 u_input,
		map_on:                  false,
		view_id:                 -1,
		stick_id:                -1,
		allocated_id:            []bool{false, false, false, false, false, false, false, false, false, false},
		config:                  config_json,
		joystickInfo:            joystickInfo,
		screen_x:                screen_x,
		screen_y:                screen_y,
		screen_init_x:           int(screen_x/2 + 100),
		screen_init_y:           int(screen_y / 2),
		screen_current_x:        int(screen_x/2 + 100),
		screen_current_y:        int(screen_y / 2),
		view_lock:               sync.Mutex{},
		auto_release_view_count: 0,
		abs_last: map[string]float64{
			"HAT0X": 0.5,
			"HAT0Y": 0.5,
			"LT":    0,
			"RT":    0,
			"LS_X":  0.5,
			"LS_Y":  0.5,
			"RS_X":  0.5,
			"RS_Y":  0.5,
		},
	}
}

func (self *TouchHandler) require_id() int8 {
	for i, v := range self.allocated_id {
		if !v {
			self.allocated_id[i] = true
			return int8(i)
		}
	}
	return -1
}

func (self *TouchHandler) u_input_control(action int, arg1 int, arg2 int) {
	self.u_input <- &u_input_control_pack{
		action: action,
		arg1:   arg1,
		arg2:   arg2,
	}
}

func (self *TouchHandler) touch_control(action int8, id int8, x int, y int) {
	self.touch_controller <- &touch_control_pack{
		action: action,
		id:     id,
		x:      x,
		y:      y,
	}
}

func (self *TouchHandler) handel_view_move(offset_x int, offset_y int) { //视角移动
	self.view_lock.Lock()
	self.auto_release_view_count = 0
	if self.view_id == -1 {
		self.view_id = self.require_id() //记得手动释放
		if self.view_id == -1 {
			return
		}
		self.touch_control(TouchActionRequire, self.view_id, self.screen_init_x, self.screen_init_y)
		self.screen_current_x = self.screen_init_x
		self.screen_current_y = self.screen_init_y
	}
	self.screen_current_x += offset_x
	self.screen_current_y += offset_y
	if false { //有界 or 无界 即 使用eventX 还是 inputManager
		if self.screen_current_x <= 0 || self.screen_current_x >= self.screen_x || self.screen_current_y <= 0 || self.screen_current_y >= self.screen_y {
			fmt.Printf("out of screen\n")
			self.touch_control(TouchActionRelease, self.view_id, -1, -1)
			self.allocated_id[self.view_id] = false
			self.view_id = self.require_id()
			self.touch_control(TouchActionRequire, self.view_id, self.screen_init_x, self.screen_init_y)
			self.screen_current_x = self.screen_init_x + offset_x
			self.screen_current_y = self.screen_init_y + offset_y
		}
	}
	self.touch_control(TouchActionMove, self.view_id, self.screen_current_x, self.screen_current_y)
	self.view_lock.Unlock()
}

func (self *TouchHandler) auto_handel_view_release() { //视角释放
	for {
		if self.view_id != -1 {
			atomic.AddInt32(&self.auto_release_view_count, 1)
			if self.auto_release_view_count > 10 { //一秒钟不动 则释放
				self.view_lock.Lock()
				self.auto_release_view_count = 0
				self.allocated_id[self.view_id] = false
				self.view_id = -1
				self.touch_control(TouchActionRelease, self.view_id, -1, -1)
				self.view_lock.Unlock()
			}
		}
		time.Sleep(time.Duration(100) * time.Millisecond)
	}
}

func (self *TouchHandler) handel_rel_event(x int, y int, whx int, why int) {
	if x != 0 || y != 0 {
		if self.map_on {
			self.handel_view_move(x, y)
		} else {
			self.u_input_control(UInput_mouse_move, x, y)
		}
	}
}

func (self *TouchHandler) handel_key_up_down(key_name string, upd_own int32) {
	if key_name == "" {
		return
	}

	if upd_own == DOWN {
		fmt.Printf("key: %s down\n ", key_name)
	} else {
		fmt.Printf("key: %s up\n ", key_name)
	}
}

func (self *TouchHandler) handel_key_events(events []*evdev.Event, dev_name string) {
	if self.joystickInfo[dev_name] != nil {
		for _, event := range events {
			key_name := self.joystickInfo[dev_name].Get("BTN").Get(strconv.Itoa(int(event.Code))).MustString("")
			self.handel_key_up_down(key_name, event.Value)
		}
	} else {
		for _, event := range events {
			self.handel_key_up_down(GetKeyName(event.Code), event.Value)
		}
	}
}

func (self *TouchHandler) handel_abs_events(events []*evdev.Event, dev_name string) {
	for _, event := range events {
		if self.joystickInfo[dev_name] != nil {
			abs_info := self.joystickInfo[dev_name].Get("ABS").Get(strconv.Itoa(int(event.Code)))
			name := abs_info.Get("name").MustString("")
			abs_mini := int32(abs_info.Get("range").GetIndex(0).MustInt())
			abs_max := int32(abs_info.Get("range").GetIndex(1).MustInt())
			formated_value := float64(event.Value-abs_mini) / float64(abs_max-abs_mini)

			if name == "HAT0X" || name == "HAT0Y" {
				down_up_key := fmt.Sprintf("%s_%s", strconv.FormatFloat(self.abs_last[name], 'f', 1, 64), strconv.FormatFloat(formated_value, 'f', 1, 64))
				self.abs_last[name] = formated_value
				direction := HAT_D_U[down_up_key][0]
				up_down := HAT_D_U[down_up_key][1]
				translated_name := HAT0_KEYNAME[name][direction]
				self.handel_key_up_down(translated_name, up_down)
			} else if name == "LT" || name == "RT" {
				for i := 0; i < 6; i++ {
					if self.abs_last[name] < float64(i)/5 && formated_value >= float64(i)/5 {
						translated_name := fmt.Sprintf("%s_%d", name, i)
						self.handel_key_up_down(translated_name, DOWN)
					} else if self.abs_last[name] >= float64(i)/5 && formated_value < float64(i)/5 {
						translated_name := fmt.Sprintf("%s_%d", name, i)
						self.handel_key_up_down(translated_name, UP)
					}
				}
				self.abs_last[name] = formated_value
			} else { //必定摇杆
				self.abs_last[name] = formated_value
				fmt.Printf("%s: %f\n", name, formated_value)
			}

		} else {
			fmt.Println(dev_name + " config not found")
		}

	}

}

func (self *TouchHandler) handel_event() {
	for {
		key_events := make([]*evdev.Event, 0)
		abs_events := make([]*evdev.Event, 0)
		var x int = 0
		var y int = 0
		var whx int = 0
		var why int = 0
		event_pack := <-self.events
		for _, event := range event_pack.events {
			switch event.Type {
			case evdev.EventKey:
				key_events = append(key_events, event)
			case evdev.EventAbsolute:
				abs_events = append(abs_events, event)
			case evdev.EventRelative:
				switch event.Code {
				case uint16(evdev.RelativeX):
					x = int(event.Value)
				case uint16(evdev.RelativeY):
					y = int(event.Value)
				case uint16(evdev.RelativeHWheel):
					whx = int(event.Value)
				case uint16(evdev.RelativeWheel):
					why = int(event.Value)
				}
			}
		}
		self.handel_rel_event(x, y, whx, why)
		self.handel_key_events(key_events, event_pack.dev_name)
		self.handel_abs_events(abs_events, event_pack.dev_name)
	}
}
