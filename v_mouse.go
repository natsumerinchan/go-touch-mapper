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
}

func init_v_mouse_controller(
	touchHandlerInstance *TouchHandler,
	u_input_control_ch chan *u_input_control_pack,
	fileted_u_input_control_ch chan *u_input_control_pack,
) *v_mouse_controller {
	udp_ch := make(chan []byte)
	go (func() {
		socket, err := net.DialUDP("udp", nil, &net.UDPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 6533,
		})
		if err != nil {
			fmt.Printf("连接v_mouse失败 : %s\n", err.Error())
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
	}
}

func (self *v_mouse_controller) main_loop() {
	for {
		select {
		case <-global_close_signal:
			return
		default:
			data := <-self.uinput_in
			fmt.Printf("uinput:%v\n", data)
			self.uinput_out <- data
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
		self.display_mouse_control(true, self.left_downing, self.mouse_x, self.mouse_y)
		if self.left_downing && self.mouse_id != -1 {
			self.touchHandlerInstance.touch_move(self.mouse_id, self.mouse_x, self.mouse_y)
		}
	} else {
		fmt.Printf("ERROR: mouse_move: not working\n")
	}
}

func (self *v_mouse_controller) on_left_btn(up_down int32) {
	if self.working {
		if up_down == DOWN {
			self.left_downing = true
			self.mouse_id = self.touchHandlerInstance.touch_require(self.mouse_x, self.mouse_y)
		} else {
			self.left_downing = false
			self.touchHandlerInstance.touch_release(self.mouse_id)
		}
	} else {
		fmt.Printf("ERROR: left_btn: not working\n")
	}
}

func (self *v_mouse_controller) on_hwheel_action(value int32) { //有待优化
	if self.working {
		hwheel_id := self.touchHandlerInstance.touch_require(self.mouse_x, self.mouse_y)
		self.touchHandlerInstance.touch_move(hwheel_id, self.mouse_x, self.mouse_y+value)
		self.touchHandlerInstance.touch_release(hwheel_id)
	} else {
		fmt.Printf("ERROR: hwheel_action: not working\n")
	}
}

func (self *v_mouse_controller) on_map_switch() { //map_on 切换
	self.working = !self.working
	if self.working {
		self.display_mouse_control(true, self.left_downing, self.mouse_x, self.mouse_y)
	} else {
		self.display_mouse_control(false, false, 0, 0)
	}
}
