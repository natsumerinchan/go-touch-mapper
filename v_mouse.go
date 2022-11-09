package main

import (
	"fmt"
	"net"
	"os"
)

type v_mouse_controller struct {
	touchHandlerInstance *TouchHandler
	uinput_in            chan *u_input_control_pack
	uinput_out           chan *u_input_control_pack
	working              bool
	left_downing         bool
	mouse_x              int32
	mouse_y              int32
	udp_write_ch         chan []byte
	mouse_id             int32
	screen_x             int32
	screen_y             int32
	map_switch_signal    chan bool
}

func init_v_mouse_controller(
	touchHandlerInstance *TouchHandler,
	u_input_control_ch chan *u_input_control_pack,
	fileted_u_input_control_ch chan *u_input_control_pack,
	map_switch_signal chan bool,
) *v_mouse_controller {
	udp_ch := make(chan []byte)
	go (func() {
		socket, err := net.DialUDP("udp", nil, &net.UDPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 6533,
		})
		if err != nil {
			logger.Errorf("连接v_mouse失败 : %s", err.Error())
			os.Exit(3)
		}
		defer socket.Close()
		for {
			select {
			case <-global_close_signal:
				return
			default:
				data := <-udp_ch
				socket.Write(data)
			}
		}
	})()
	screen_x, screen_y := get_wm_size()

	return &v_mouse_controller{
		touchHandlerInstance: touchHandlerInstance,
		uinput_in:            u_input_control_ch,
		uinput_out:           fileted_u_input_control_ch,
		working:              true,
		left_downing:         false,
		mouse_x:              0,
		mouse_y:              0,
		udp_write_ch:         udp_ch,
		mouse_id:             -1,
		screen_x:             screen_x,
		screen_y:             screen_y,
		map_switch_signal:    map_switch_signal,
	}
}

func (self *v_mouse_controller) get_max_xy_val() (int32, int32) {
	if global_device_orientation == 0 || global_device_orientation == 2 {
		return self.screen_x, self.screen_y
	} else {
		return self.screen_y, self.screen_x
	}
}

func (self *v_mouse_controller) main_loop() {
	for {
		select {
		case <-global_close_signal:
			return
		case <-self.map_switch_signal:
			self.working = !self.working
			if self.working {
				self.display_mouse_control(true, self.left_downing, self.mouse_x, self.mouse_y)
			} else {
				self.display_mouse_control(false, false, 0, 0)
			}
		case data := <-self.uinput_in:
			if data.action == UInput_mouse_move {
				self.on_mouse_move(data.arg1, data.arg2)
			} else {
				if data.arg1 >= 0x110 && data.arg1 <= 0x117 {
					if data.arg1 == 0x110 {
						self.on_left_btn(data.arg2)
					}
				} else {
					self.uinput_out <- data
				}
			}
		}
	}
}

func (self *v_mouse_controller) display_mouse_control(show, downing bool, abs_x, abs_y int32) { //控制鼠标图标
	var show_int int32
	if show {
		show_int = 1
	} else {
		show_int = 0
	}
	var downing_int int32
	if downing {
		downing_int = 1
	} else {
		downing_int = 0
	}
	fmt_str := fmt.Sprintf("%d,%d,%d,%d", abs_x, abs_y, show_int, downing_int)
	self.udp_write_ch <- []byte(fmt_str)
}

func (self *v_mouse_controller) on_mouse_move(rel_x, rel_y int32) {
	if self.working {
		self.mouse_x += rel_x
		self.mouse_y += rel_y
		max_x, max_y := self.get_max_xy_val()
		if self.mouse_x < 0 {
			self.mouse_x = 0
		}
		if self.mouse_y < 0 {
			self.mouse_y = 0
		}
		if self.mouse_x > max_x {
			self.mouse_x = max_x
		}
		if self.mouse_y > max_y {
			self.mouse_y = max_y
		}
		self.display_mouse_control(true, self.left_downing, self.mouse_x, self.mouse_y)
		if self.left_downing && self.mouse_id != -1 {
			self.touchHandlerInstance.touch_move(self.mouse_id, self.mouse_x, self.mouse_y, touch_pos_scale)
		}
	} else {
		logger.Error("ERROR: mouse_move: not working")
	}
}

func (self *v_mouse_controller) on_left_btn(up_down int32) {
	if self.working {
		if up_down == DOWN {
			self.left_downing = true
			self.mouse_id = self.touchHandlerInstance.touch_require(self.mouse_x, self.mouse_y, touch_pos_scale)
		} else {
			self.left_downing = false
			self.touchHandlerInstance.touch_release(self.mouse_id)
		}
		self.on_mouse_move(0, 0)
	} else {
		logger.Error("ERROR: left_btn: not working")
	}
}

func (self *v_mouse_controller) on_hwheel_action(value int32) { //有待优化
	if self.working {
		hwheel_id := self.touchHandlerInstance.touch_require(self.mouse_x, self.mouse_y, touch_pos_scale)
		self.touchHandlerInstance.touch_move(hwheel_id, self.mouse_x, self.mouse_y+value, touch_pos_scale)
		self.touchHandlerInstance.touch_release(hwheel_id)
	} else {
		logger.Error("ERROR: hwheel_action: not working")
	}
}
