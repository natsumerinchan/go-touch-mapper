package main

import (
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
