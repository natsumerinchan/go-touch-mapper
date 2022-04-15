package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"sync"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/kenshaw/evdev"
)

type TouchHandler struct {
	events                  chan *event_pack            //接收事件的channel
	touch_controll_channel  chan *touch_control_pack    //发送触屏控制信号的channel
	u_input                 chan *u_input_control_pack  //发送u_input控制信号的channel
	map_on                  bool                        //映射模式开关
	view_id                 int32                       //视角的触摸ID
	wheel_id                int32                       //左摇杆的触摸ID
	allocated_id            []bool                      //10个触摸点分配情况
	config                  *simplejson.Json            //映射配置文件
	joystickInfo            map[string]*simplejson.Json //所有摇杆配置文件 dev_name 为key
	screen_x                int32                       //屏幕宽度
	screen_y                int32                       //屏幕高度
	view_init_x             int32                       //初始化视角映射的x坐标
	view_init_y             int32                       //初始化视角映射的y坐标
	view_current_x          int32                       //当前视角映射的x坐标
	view_current_y          int32                       //当前视角映射的y坐标
	view_speed_x            int32                       //视角x方向的速度
	view_speed_y            int32                       //视角y方向的速度
	wheel_init_x            int32                       //初始化左摇杆映射的x坐标
	wheel_init_y            int32                       //初始化左摇杆映射的y坐标
	wheel_range             int32                       //左摇杆的x轴范围
	view_lock               sync.Mutex                  //视角控制相关的锁 用于自动释放和控制相关
	wheel_lock              sync.Mutex                  //左摇杆控制相关的锁 用于自动释放和控制相关
	auto_release_view_count int32                       //自动释放计时器 有视角移动则重置 否则100ms加一 超过1s 自动释放
	abs_last                sync.Map                    //abs值的上一次值 用于手柄
	using_joystick_name     string                      //当前正在使用的手柄 针对不同手柄死区不同 但程序支持同时插入多个手柄 因此会识别最进发送事件的手柄作为死区配置
	ls_wheel_released       bool                        //左摇杆滚轮释放
	wasd_wheel_released     bool                        //wasd滚轮释放 两个都释放时 轮盘才会释放
	wasd_wheel_last_x       int32                       //wasd滚轮上一次的x坐标
	wasd_wheel_last_y       int32                       //wasd滚轮上一次的y坐标
	wasd_up_down_stause     []bool
	key_action_state_save   sync.Map
}

const (
	TouchActionRequire int8 = 0
	TouchActionRelease int8 = 1
	TouchActionMove    int8 = 2
	TouchActionSync    int8 = 3
)

const (
	UInput_mouse_move int8 = 0
	UInput_mouse_btn  int8 = 1
	UInput_key_event  int8 = 2
)

const (
	DOWN int32 = 1
	UP   int32 = 0
)

const (
	Wheel_action_move    int8 = 1
	Wheel_action_release int8 = 0
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
	// fmt.Printf("joystickInfo:%v\n", joystickInfo)

	abs_last_map := sync.Map{}

	abs_last_map.Store("HAT0X", 0.5)
	abs_last_map.Store("HAT0Y", 0.5)
	abs_last_map.Store("LT", 0.0)
	abs_last_map.Store("RT", 0.0)
	abs_last_map.Store("LS_X", 0.5)
	abs_last_map.Store("LS_Y", 0.5)
	abs_last_map.Store("RS_X", 0.5)
	abs_last_map.Store("RS_Y", 0.5)

	return &TouchHandler{
		events:                 events,
		touch_controll_channel: touch_controller,
		u_input:                u_input,
		map_on:                 true, //false
		view_id:                -1,
		wheel_id:               -1,
		allocated_id:           []bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
		// ^^^ 是可以创建超过12个的 只是不显示白点罢了
		config:                  config_json,
		joystickInfo:            joystickInfo,
		screen_x:                int32(config_json.Get("SCREEN").Get("SIZE").GetIndex(0).MustInt()),
		screen_y:                int32(config_json.Get("SCREEN").Get("SIZE").GetIndex(1).MustInt()),
		view_init_x:             int32(config_json.Get("MOUSE").Get("POS").GetIndex(0).MustInt()),
		view_init_y:             int32(config_json.Get("MOUSE").Get("POS").GetIndex(1).MustInt()),
		view_current_x:          int32(config_json.Get("MOUSE").Get("POS").GetIndex(0).MustInt()),
		view_current_y:          int32(config_json.Get("MOUSE").Get("POS").GetIndex(1).MustInt()),
		view_speed_x:            int32(config_json.Get("MOUSE").Get("SPEED").GetIndex(0).MustInt()),
		view_speed_y:            int32(config_json.Get("MOUSE").Get("SPEED").GetIndex(1).MustInt()),
		wheel_init_x:            int32(config_json.Get("WHEEL").Get("POS").GetIndex(0).MustInt()),
		wheel_init_y:            int32(config_json.Get("WHEEL").Get("POS").GetIndex(1).MustInt()),
		wheel_range:             int32(config_json.Get("WHEEL").Get("RANGE").MustInt()),
		view_lock:               sync.Mutex{},
		wheel_lock:              sync.Mutex{},
		auto_release_view_count: 0,
		abs_last:                abs_last_map,
		using_joystick_name:     "",
		ls_wheel_released:       true,
		wasd_wheel_released:     true,
		wasd_wheel_last_x:       int32(config_json.Get("WHEEL").Get("POS").GetIndex(0).MustInt()),
		wasd_wheel_last_y:       int32(config_json.Get("WHEEL").Get("POS").GetIndex(1).MustInt()),
		wasd_up_down_stause:     []bool{false, false, false, false},
		key_action_state_save:   sync.Map{},
	}
}

func (self *TouchHandler) touch_sync() {
	self.send_touch_control_pack(TouchActionSync, -1, -1, -1)
}

func (self *TouchHandler) touch_require(x int32, y int32) int32 {
	for i, v := range self.allocated_id {
		if !v {
			self.allocated_id[i] = true
			self.send_touch_control_pack(TouchActionRequire, int32(i), x, y)
			// self.touch_sync()
			return int32(i)
		}
	}
	return -1
}

func (self *TouchHandler) touch_release(id int32) {
	if id != -1 {
		self.allocated_id[int(id)] = false
		self.send_touch_control_pack(TouchActionRelease, id, -1, -1)
		// self.touch_sync()
	}
}

func (self *TouchHandler) touch_move(id int32, x int32, y int32) {
	if id != -1 {
		self.send_touch_control_pack(TouchActionMove, id, x, y)
		// self.touch_sync()
	}
}

func (self *TouchHandler) u_input_control(action int8, arg1 int32, arg2 int32) {
	self.u_input <- &u_input_control_pack{
		action: action,
		arg1:   arg1,
		arg2:   arg2,
	}
}

func (self *TouchHandler) send_touch_control_pack(action int8, id int32, x int32, y int32) {
	self.touch_controll_channel <- &touch_control_pack{
		action:   action,
		id:       id,
		x:        x,
		y:        y,
		screen_x: self.screen_x,
		screen_y: self.screen_y,
	}
}

func (self *TouchHandler) loop_handel_rs_move() {
	for {
		// fmt.Printf("rs: %f , %f\n", self.abs_last["RS_X"], self.abs_last["RS_Y"])
		// rs_x := self.abs_last["RS_X"]
		// rs_y := self.abs_last["RS_Y"]
		rs_x, rs_y := self.getStick("RS")
		if rs_x != 0.5 || rs_y != 0.5 {
			if self.map_on {
				self.handel_view_move(int32((rs_x-0.5)*24), int32((rs_y-0.5)*24))
			} else {
				self.u_input_control(UInput_mouse_move, int32((rs_x-0.5)*24), int32((rs_y-0.5)*24))
			}
		}
		time.Sleep(time.Duration(10) * time.Millisecond)
	}
}

func (self *TouchHandler) handel_view_move(offset_x int32, offset_y int32) { //视角移动
	self.view_lock.Lock()
	self.auto_release_view_count = 0
	if self.view_id == -1 {
		// self.view_id = self.require_id() //记得手动释放
		// if self.view_id == -1 {
		// 	return
		// }
		// self.touch_control(TouchActionRequire, self.view_id, self.view_init_x, self.view_init_y)
		self.view_id = self.touch_require(self.view_init_x, self.view_init_y)
		self.view_current_x = self.view_init_x
		self.view_current_y = self.view_init_y
	}
	self.view_current_x -= offset_y * self.view_speed_y //用的时直接写event坐标系
	self.view_current_y += offset_x * self.view_speed_x
	if true { //有界 or 无界 即 使用eventX 还是 inputManager
		if self.view_current_x <= 0 || self.view_current_x >= self.screen_x || self.view_current_y <= 0 || self.view_current_y >= self.screen_y {
			fmt.Printf("out of screen\n")
			// self.touch_control(TouchActionRelease, self.view_id, -1, -1)
			// self.touch_control(TouchActionRequire, self.view_id, self.view_init_x, self.view_init_y)
			self.touch_release(self.view_id)
			self.view_id = self.touch_require(self.view_init_x, self.view_init_y)
			self.view_current_x = self.view_init_x - offset_y*self.view_speed_y
			self.view_current_y = self.view_init_y + offset_x*self.view_speed_x
		}
	}
	// self.send_touch_control_pack(TouchActionMove, self.view_id, self.view_current_x, self.view_current_y)
	self.touch_move(self.view_id, self.view_current_x, self.view_current_y)
	self.view_lock.Unlock()
}

func (self *TouchHandler) auto_handel_view_release() { //视角释放
	for {
		self.view_lock.Lock()
		if self.view_id != -1 {
			self.auto_release_view_count += 1
			if self.auto_release_view_count > 10 { //一秒钟不动 则释放
				self.auto_release_view_count = 0
				self.allocated_id[self.view_id] = false
				// self.send_touch_control_pack(TouchActionRelease, self.view_id, -1, -1)
				self.touch_release(self.view_id)
				self.view_id = -1
			}
		}
		self.view_lock.Unlock()
		time.Sleep(time.Duration(20) * time.Millisecond)
	}
}

func (self *TouchHandler) handel_wheel_action(action int8, abs_x int32, abs_y int32) {
	self.wheel_lock.Lock()
	if action == Wheel_action_release { //释放
		if self.wheel_id != -1 {
			// self.send_touch_control_pack(TouchActionRelease, self.wheel_id, -1, -1)
			self.touch_release(self.wheel_id)
			// fmt.Printf("wheel release  %d -> -1\n", self.wheel_id)
			self.allocated_id[self.wheel_id] = false
			self.wheel_id = -1
		}
	} else if action == Wheel_action_move { //移动
		if self.wheel_id == -1 { //如果在移动之前没有按下
			// self.wheel_id = self.require_id()                                                           //申请id
			// self.touch_control(TouchActionRequire, self.wheel_id, self.wheel_init_x, self.wheel_init_y) //按下中心
			self.wheel_id = self.touch_require(self.wheel_init_x, self.wheel_init_y)
		}
		// self.touch_control(TouchActionMove, self.wheel_id, abs_x, abs_y) //移动
		self.touch_move(self.wheel_id, abs_x, abs_y)
	}
	self.wheel_lock.Unlock()
}

func (self *TouchHandler) get_wasd_now_target() (int32, int32) { //根据wasd当前状态 获取wasd滚轮的目标位置
	return 300, 400
}

func (self *TouchHandler) loop_handel_wasd_wheel() { //循环处理wasd映射轮盘并控制释放
	for {
		wasd_wheel_target_x, wasd_wheel_target_y := self.get_wasd_now_target() //获取目标位置
		if self.wheel_init_x == wasd_wheel_target_x && self.wheel_init_y == wasd_wheel_target_y {
			self.wasd_wheel_released = true //如果wasd目标位置 等于 wasd轮盘初始位置 则认为轮盘释放
		} else {
			if self.wasd_wheel_last_x != wasd_wheel_target_x || self.wasd_wheel_last_y != wasd_wheel_target_y {
				//否则如果目标位置不等于当前位置
				//则步进移动到目标位置 同时更新当前位置
			}
		}
		if self.wheel_id != -1 && self.wasd_wheel_released && self.ls_wheel_released {
			self.handel_wheel_action(Wheel_action_release, -1, -1) //wheel当前按下 且两个标记都释放 则释放
		}
	}
}

func (self *TouchHandler) handel_rel_event(x int32, y int32, whx int32, why int32) {
	if x != 0 || y != 0 {
		if self.map_on {
			self.handel_view_move(x, y)
		} else {
			self.u_input_control(UInput_mouse_move, x, y)
		}
	}
}

func (self *TouchHandler) excute_key_action(key_name string, up_down int32, action *simplejson.Json, state interface{}) {
	// fmt.Printf("excute action up_down:%d , action %v , state %v\n", up_down, action, state == nil)
	switch action.Get("TYPE").MustString() {
	case "PRESS": //按键的按下与释放直接映射为触屏的按下与释放
		if up_down == DOWN {
			x := int32(action.Get("POS").GetIndex(0).MustInt())
			y := int32(action.Get("POS").GetIndex(1).MustInt())
			// tid := self.require_id()
			// self.touch_control(TouchActionRequire, tid, x, y)
			self.key_action_state_save.Store(key_name, self.touch_require(x, y))
		} else if up_down == UP {
			tid := state.(int32)
			// self.touch_control(TouchActionRelease, tid, -1, -1)
			// self.allocated_id[tid] = false
			self.touch_release(tid)
			self.key_action_state_save.Delete(key_name)
		}
	case "CLICK": //仅在按下的时候执行一次 不保存状态所以不响应down 也不会有down到这里
		if up_down == DOWN {
			x := int32(action.Get("POS").GetIndex(0).MustInt())
			y := int32(action.Get("POS").GetIndex(1).MustInt())
			// tid := self.require_id()
			// self.touch_control(TouchActionRequire, tid, x, y)
			tid := self.touch_require(x, y)
			time.Sleep(time.Duration(8) * time.Millisecond) //8ms 120HZ下一次
			// self.touch_control(TouchActionRelease, tid, -1, -1)
			// self.allocated_id[tid] = false
			self.touch_release(tid)
		}

	case "AUTO_FIRE": //连发 按下开始 松开结束 按照设置的间隔 持续点击
		if up_down == DOWN {
			x := int32(action.Get("POS").GetIndex(0).MustInt())
			y := int32(action.Get("POS").GetIndex(1).MustInt())
			down_time := action.Get("INTERVAL").GetIndex(0).MustInt()
			interval_time := action.Get("INTERVAL").GetIndex(1).MustInt()
			self.key_action_state_save.Store(key_name, true)
			for {
				if running, _ := self.key_action_state_save.Load(key_name); running == true {
					// self.touch_control(TouchActionRequire, tid, x, y)
					tid := self.touch_require(x, y)
					time.Sleep(time.Duration(down_time) * time.Millisecond)
					// self.touch_control(TouchActionRelease, tid, -1, -1)
					self.touch_release(tid)
					time.Sleep(time.Duration(interval_time) * time.Millisecond)
				} else {
					break
				}
			}
			// self.allocated_id[tid] = false
			self.key_action_state_save.Delete(key_name)
		} else if up_down == UP {
			self.key_action_state_save.Store(key_name, false)
		}

	case "MULT_PRESS": //多点触摸 按照顺序按下 松开再反向松开 实现类似一键开镜开火
		if up_down == DOWN {
			tid_save := make([]int32, 0)
			release_signal := make(chan bool, 16)
			self.key_action_state_save.Store(key_name, release_signal)
			for i := range action.Get("POS_S").MustArray() {
				x := int32(action.Get("POS_S").GetIndex(i).GetIndex(0).MustInt())
				y := int32(action.Get("POS_S").GetIndex(i).GetIndex(1).MustInt())
				// tid := self.require_id()
				// self.touch_control(TouchActionRequire, tid, x, y)
				tid := self.touch_require(x, y)
				tid_save = append(tid_save, tid)
				time.Sleep(time.Duration(8) * time.Millisecond) // 间隔8ms 是否需要延迟有待验证
			}
			<-release_signal
			self.key_action_state_save.Delete(key_name)
			for i := len(tid_save) - 1; i >= 0; i-- {
				// self.touch_control(TouchActionRelease, tid_save[i], -1, -1)
				// if tid_save[i] != -1 {
				// 	self.allocated_id[tid_save[i]] = false
				// }
				self.touch_release(tid_save[i])
				time.Sleep(time.Duration(8) * time.Millisecond)
			}
		} else if up_down == UP {
			state.(chan bool) <- true
			//按下立即创建channel 并保存状态
			//松开拿到的channel 并发送信号
			//同时立即删除状态
			//即按下立即执行并等待释放,此过程中不响应按下但是可以响应多次松开 //缓冲区大小
			//而再松开后释放触摸过程中便可以再次响应按下
		}
	case "DRAG": //只响应一次按下  可同时多次触发
		if up_down == DOWN {

			pos_len := len(action.Get("POS_S").MustArray())
			interval_time := action.Get("INTERVAL").GetIndex(0).MustInt()
			fmt.Printf("pos_len:%d, interval_time:%d\n", pos_len, interval_time)
			init_x := int32(action.Get("POS_S").GetIndex(0).GetIndex(0).MustInt())
			init_y := int32(action.Get("POS_S").GetIndex(0).GetIndex(1).MustInt())
			// self.touch_control(TouchActionRequire, tid, init_x, init_y)
			tid := self.touch_require(init_x, init_y)
			time.Sleep(time.Duration(interval_time) * time.Millisecond)
			for index := 1; index < pos_len-1; index++ {
				x := int32(action.Get("POS_S").GetIndex(index).GetIndex(0).MustInt())
				y := int32(action.Get("POS_S").GetIndex(index).GetIndex(1).MustInt())
				// self.touch_control(TouchActionMove, tid, x, y)
				self.touch_move(tid, x, y)
				time.Sleep(time.Duration(interval_time) * time.Millisecond)
			}
			end_x := int32(action.Get("POS_S").GetIndex(pos_len - 1).GetIndex(0).MustInt())
			end_y := int32(action.Get("POS_S").GetIndex(pos_len - 1).GetIndex(1).MustInt())
			self.touch_move(tid, end_x, end_y)
			self.touch_release(tid)
			// self.touch_control(TouchActionMove, tid, end_x, end_y)
			// self.touch_control(TouchActionRelease, tid, -1, -1)
			// self.allocated_id[tid] = false
		} else if up_down == UP {

		}

	}
}

func (self *TouchHandler) handel_key_up_down(key_name string, up_down int32, dev_name string) {
	// fmt.Printf("key_name:%s, upd_own:%d, dev_name:%s\n", key_name, up_down, dev_name)
	if key_name == "" {
		return
	}

	if self.map_on {
		if action, ok := self.config.Get("KEY_MAPS").CheckGet(key_name); ok {
			// fmt.Printf("key_name:%s, action:%+v\n", key_name, action)
			state, ok := self.key_action_state_save.Load(key_name)
			if (up_down == UP && !ok) || (up_down == DOWN && ok) {
				// fmt.Printf("duplicate or not down key:%s\n", key_name)
			} else {
				go self.excute_key_action(key_name, up_down, action, state)
			}
		} else {
			// fmt.Printf("key_name:%s, not mapped\n", key_name)
			return
		}
	} else {
		//映射关时 输出u_input按键
		if jsconfig, ok := self.joystickInfo[dev_name]; ok {
			//如果是手柄 则检查是否设置了键盘映射
			if joystick_btn_map_key_name, ok := jsconfig.Get("MAP_KEYBOARD").CheckGet(key_name); ok {
				//有则映射到普通按键
				self.handel_key_up_down(joystick_btn_map_key_name.MustString(), up_down, dev_name+"_joystick_mapped")
			} else {
				// fmt.Printf("%s key %s not set keyboard map\n", dev_name, key_name)
			}
		} else {
			if code, ok := friendly_name_2_keycode[key_name]; ok {
				//是合法按键 则输出
				self.u_input_control(UInput_key_event, int32(code), int32(up_down))
			}
		}
	}

}

func (self *TouchHandler) handel_key_events(events []*evdev.Event, dev_name string) {
	if jsconfig, ok := self.joystickInfo[dev_name]; ok {
		for _, event := range events {
			if key_name, ok := jsconfig.Get("BTN").CheckGet(strconv.Itoa(int(event.Code))); ok {
				self.handel_key_up_down(key_name.MustString(), event.Value, dev_name)
			} else {
				// fmt.Printf("unknown code %d from %s \n", event.Code, dev_name)
			}
		}
	} else {
		for _, event := range events {
			self.handel_key_up_down(GetKeyName(event.Code), event.Value, dev_name)
		}
	}
}

func (self *TouchHandler) getStick(stick_name string) (float64, float64) {
	if jsconfig, ok := self.joystickInfo[self.using_joystick_name]; ok {
		_x, _ := self.abs_last.Load(stick_name + "_X")
		_y, _ := self.abs_last.Load(stick_name + "_Y")
		x, y := _x.(float64), _y.(float64)
		deadZone_left := jsconfig.Get("DEADZONE").Get(stick_name).GetIndex(0).MustFloat64()
		deadZone_right := jsconfig.Get("DEADZONE").Get(stick_name).GetIndex(1).MustFloat64()
		if deadZone_left < x && x < deadZone_right && deadZone_left < y && y < deadZone_right {
			return 0.5, 0.5
		} else {
			return x, y
		}
	} else {
		return 0.5, 0.5
	}
}

func (self *TouchHandler) handel_abs_events(events []*evdev.Event, dev_name string) {
	for _, event := range events {

		if jsconfig, ok := self.joystickInfo[dev_name]; ok {
			abs_info := jsconfig.Get("ABS").Get(strconv.Itoa(int(event.Code)))
			name := abs_info.Get("name").MustString("")
			abs_mini := int32(abs_info.Get("range").GetIndex(0).MustInt())
			abs_max := int32(abs_info.Get("range").GetIndex(1).MustInt())
			formated_value := float64(event.Value-abs_mini) / float64(abs_max-abs_mini)
			_last_value, _ := self.abs_last.Load(name)
			last_value := _last_value.(float64)
			if name == "HAT0X" || name == "HAT0Y" {
				down_up_key := fmt.Sprintf("%s_%s", strconv.FormatFloat(last_value, 'f', 1, 64), strconv.FormatFloat(formated_value, 'f', 1, 64))
				self.abs_last.Store(name, formated_value)
				direction := HAT_D_U[down_up_key][0]
				up_down := HAT_D_U[down_up_key][1]
				translated_name := HAT0_KEYNAME[name][direction]
				self.handel_key_up_down(translated_name, up_down, dev_name)
			} else if name == "LT" || name == "RT" {
				for i := 0; i < 6; i++ {
					if last_value < float64(i)/5 && formated_value >= float64(i)/5 {
						translated_name := fmt.Sprintf("%s_%d", name, i)
						self.handel_key_up_down("BTN_"+translated_name, DOWN, dev_name)
					} else if last_value >= float64(i)/5 && formated_value < float64(i)/5 {
						translated_name := fmt.Sprintf("%s_%d", name, i)
						self.handel_key_up_down("BTN_"+translated_name, UP, dev_name)
					}
				}
				self.abs_last.Store(name, formated_value)
			} else { //必定摇杆
				if self.using_joystick_name != dev_name {
					self.using_joystick_name = dev_name
				}
				// self.abs_last_set(name, formated_value)
				self.abs_last.Store(name, formated_value)
				//右摇杆控制视角 只需修改值 有单独线程去处理
				//左摇杆控制轮盘 且与WASD可同时工作 在这里处理
				if (name == "LS_X" || name == "LS_Y") && self.map_on {
					ls_x, ls_y := self.getStick("LS")
					if ls_x == 0.5 && ls_y == 0.5 {
						if self.ls_wheel_released == false {
							// fmt.Println("摇杆控制轮盘已释放")
							self.ls_wheel_released = true
						}
					} else {
						self.ls_wheel_released = false
						target_x := self.wheel_init_x - int32(float64(self.wheel_range)*2*(ls_y-0.5)) //注意这里的X和Y是相反的
						target_y := self.wheel_init_y + int32(float64(self.wheel_range)*2*(ls_x-0.5))
						self.handel_wheel_action(Wheel_action_move, target_x, target_y)
					}
				}
			}
		} else {
			fmt.Println(dev_name + " config not found")
		}

	}

}

func (self *TouchHandler) mix_touch(touch_events chan *event_pack) {
	id_2_vid := make([]int32, 10) //硬件ID到虚拟ID的映射
	var x int32 = -1
	var y int32 = -1
	var last_id int32 = 0
	pos_s := make([][]int32, 10)
	for i := 0; i < 10; i++ {
		pos_s[i] = make([]int32, 2)
	}
	id_stause := make([]bool, 10)
	for i := 0; i < 10; i++ {
		id_stause[i] = false
	}
	for {
		event_pack := <-touch_events
		copy_pos_s := make([][]int32, 10)
		copy(copy_pos_s, pos_s)
		copy_id_stause := make([]bool, 10)
		copy(copy_id_stause, id_stause)
		for _, event := range event_pack.events {
			switch event.Code {
			case ABS_MT_POSITION_X:
				x = event.Value
				pos_s[last_id] = []int32{x, y}
			case ABS_MT_POSITION_Y:
				y = event.Value
				pos_s[last_id] = []int32{x, y}
			case ABS_MT_TRACKING_ID:
				if event.Value == -1 {
					id_stause[last_id] = false
				} else {
					id_stause[last_id] = true
				}
			case ABS_MT_SLOT:
				last_id = event.Value
			}
		}
		for i := 0; i < 10; i++ {
			if copy_id_stause[i] != id_stause[i] {
				if id_stause[i] { //false -> true 申请
					id_2_vid[i] = self.touch_require(pos_s[i][0], pos_s[i][1])
				} else {
					self.touch_release(id_2_vid[i])
				}
			} else {
				if pos_s[i][0] != copy_pos_s[i][0] || pos_s[i][1] != copy_pos_s[i][1] {
					self.touch_move(id_2_vid[i], pos_s[i][0], pos_s[i][1])
					fmt.Printf("%v:[%d,%d] \n", id_2_vid[i], pos_s[i][0], pos_s[i][1])
				}
			}
		}
		// for i := 0; i < 3; i++ {
		// 	fmt.Printf("%v:[%d,%d] ", id_stause[i], pos_s[i][0], pos_s[i][1])
		// }
		self.touch_sync()
		fmt.Println()

	}
}

func (self *TouchHandler) handel_event() {
	for {
		key_events := make([]*evdev.Event, 0)
		abs_events := make([]*evdev.Event, 0)
		var x int32 = 0
		var y int32 = 0
		var whx int32 = 0
		var why int32 = 0
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
					x = int32(event.Value)
				case uint16(evdev.RelativeY):
					y = int32(event.Value)
				case uint16(evdev.RelativeHWheel):
					whx = int32(event.Value)
				case uint16(evdev.RelativeWheel):
					why = int32(event.Value)
				}
			}
		}
		self.handel_rel_event(x, y, whx, why)
		self.handel_key_events(key_events, event_pack.dev_name)
		self.handel_abs_events(abs_events, event_pack.dev_name)
	}
}

func (self *TouchHandler) stop() {

}

// case ABS_MT_SLOT:
// 	last_id = event.Value
// 	tid, ok := id_2_vid[event.Value] //尝试从映射中获取虚拟ID
// 	if !ok {                         //如果没有 则说明是新申请的触摸点
// 		fmt.Printf("new touch id: %d\n", event.Value)
// 		tid = self.touch_require(x, y) //申请虚拟触摸
// 		id_2_vid[event.Value] = tid    //保存
// 		flag = false
// 	} else {
// 		flag = true
// 		//是已知的 即切换事件 会在最后处理
// 	}
// case ABS_MT_TRACKING_ID:
// 	if event.Value == -1 {
// 		self.touch_release(id_2_vid[last_id])
// 		fmt.Printf("touch release %d\n", last_id)
// 		delete(id_2_vid, last_id)
// 		flag = false
// 	}
// }

// for _, event := range event_pack.events {
// 	switch event.Code {
// 	case ABS_MT_POSITION_X:
// 		fmt.Printf("ABS_MT_POSITION_X %d\n", event.Value)
// 	case ABS_MT_POSITION_Y:
// 		fmt.Printf("ABS_MT_POSITION_Y %d\n", event.Value)
// 	case ABS_MT_TRACKING_ID:
// 		fmt.Printf("ABS_MT_TRACKING_ID %d\n", event.Value)
// 	case ABS_MT_SLOT:
// 		fmt.Printf("ABS_MT_SLOT %d\n", event.Value)
// 	}
// }
