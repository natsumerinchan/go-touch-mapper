package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bitly/go-simplejson"
	"github.com/kenshaw/evdev"
)

func create_abs_rec(name string, min, max int32) *simplejson.Json {
	obj := simplejson.New()
	obj.Set("name", name)
	_range, _ := simplejson.New().Array()
	_range = append(_range, min)
	_range = append(_range, max)
	obj.Set("range", _range)
	obj.Set("reverse", false)
	return obj
}

func create_no_block_ch(dev *evdev.Evdev) chan *event_pack {
	raw := dev.Poll(context.Background())
	events := make([]*evdev.Event, 0)
	event_reader := make(chan *event_pack)
	go func() {
		for {
			event := <-raw
			if event.Type == evdev.SyncReport {
				pack := &event_pack{
					dev_name: "ignore",
					events:   events,
				}
				select {
				case event_reader <- pack:
				default:
					// logger.Infof("ignore")
				}
				events = make([]*evdev.Event, 0)
			} else {
				events = append(events, &event.Event)
			}
		}
	}()
	return event_reader
}

func get_key(pack_ch chan *event_pack) uint16 {
	for {
		event := <-pack_ch
		for _, e := range event.events {
			if e.Type == evdev.EventKey && e.Value == UP {
				return e.Code
			}
		}
	}
}

func get_abs_meet_range(abs map[evdev.AbsoluteType]evdev.Axis, pack_ch chan *event_pack, target_value float64) uint16 {
	last_value_save := make(map[uint16]int32)
	format := func(code uint16, value int32) float64 {
		min := abs[evdev.AbsoluteType(code)].Min
		max := abs[evdev.AbsoluteType(code)].Max
		return float64(value-min) / float64(max-min)
	}
	for {
		event := <-pack_ch
		for _, e := range event.events {
			if e.Type == evdev.EventAbsolute {
				last, ok := last_value_save[e.Code]
				if ok {
					if format(e.Code, e.Value) > target_value && format(e.Code, last) < target_value {
						return e.Code
					} else {
						last_value_save[e.Code] = e.Value
						// logger.Infof("%v", last_value_save)
					}
				} else {
					last_value_save[e.Code] = e.Value
				}
			}
		}
	}
}

func get_abs_map(abs map[evdev.AbsoluteType]evdev.Axis, pack_ch chan *event_pack, LT_RT_BTN bool) map[uint16]string {
	result := make(map[uint16]string)
	used := make(map[uint16]bool)
	for k, _ := range abs {
		used[uint16(k)] = false
	}
	used[16] = true
	used[17] = true
	if !LT_RT_BTN {
		logger.Info("Pull LT down")
		for {
			code := get_abs_meet_range(abs, pack_ch, 0.99)
			if !used[code] {
				used[code] = true
				result[code] = "LT"
				break
			}
		}
		logger.Info("Pull RT down")
		for {
			code := get_abs_meet_range(abs, pack_ch, 0.99)
			if !used[code] {
				used[code] = true
				result[code] = "RT"
				break
			}
		}
	}
	for _, axis := range []string{"LS", "RS"} {
		logger.Infof("%s ↓", axis)
		for {
			code := get_abs_meet_range(abs, pack_ch, 0.99)
			if !used[code] {
				used[code] = true
				result[code] = fmt.Sprintf("%s_Y", axis)
				break
			}
		}
		logger.Infof("%s →", axis)
		for {
			code := get_abs_meet_range(abs, pack_ch, 0.99)
			if !used[code] {
				used[code] = true
				result[code] = fmt.Sprintf("%s_X", axis)
				break
			}
		}
	}
	return result
}

func create_js_info_file(index int) {
	dev_path := fmt.Sprintf("/dev/input/event%d", index)
	fd, err := os.OpenFile(dev_path, os.O_RDONLY, 0)
	if err != nil {
		logger.Errorf("打开设备文件失败, %v", err)
		return
	}
	d := evdev.Open(fd)
	defer d.Close()
	d.Lock()
	defer d.Unlock()
	pack_ch := create_no_block_ch(d)
	dev_name := d.Name()
	abs := d.AbsoluteTypes()
	keys := d.KeyTypes()
	logger.Infof("找到设备 : %s", dev_name)
	// for k, v := range abs {
	// 	logger.Infof("Absolute : %d", int(k))
	// 	logger.Infof("\t%d,%d", v.Min, v.Max)
	// }
	// for k, _ := range keys {
	// 	logger.Infof("Key : %d", int(k))
	// }

	output := simplejson.New()
	LS_DZ, _ := simplejson.New().Array()
	LS_DZ = append(LS_DZ, 0.5-0.1)
	LS_DZ = append(LS_DZ, 0.5+0.1)
	RS_DZ, _ := simplejson.New().Array()
	RS_DZ = append(RS_DZ, 0.5-0.04)
	RS_DZ = append(RS_DZ, 0.5+0.04)
	output.SetPath([]string{"DEADZONE", "LS"}, LS_DZ)
	output.SetPath([]string{"DEADZONE", "RS"}, RS_DZ)
	output.SetPath([]string{"MAP_KEYBOARD", "BTN_LT"}, "BTN_RIGHT")
	output.SetPath([]string{"MAP_KEYBOARD", "BTN_LT_2"}, "BTN_RIGHT")
	output.SetPath([]string{"MAP_KEYBOARD", "BTN_RT"}, "BTN_LEFT")
	output.SetPath([]string{"MAP_KEYBOARD", "BTN_RT_2"}, "BTN_LEFT")
	output.SetPath([]string{"MAP_KEYBOARD", "BTN_DPAD_UP"}, "KEY_UP")
	output.SetPath([]string{"MAP_KEYBOARD", "BTN_DPAD_LEFT"}, "KEY_LEFT")
	output.SetPath([]string{"MAP_KEYBOARD", "BTN_DPAD_RIGHT"}, "KEY_RIGHT")
	output.SetPath([]string{"MAP_KEYBOARD", "BTN_DPAD_DOWN"}, "KEY_DOWN")
	output.SetPath([]string{"MAP_KEYBOARD", "BTN_A"}, "KEY_ENTER")
	output.SetPath([]string{"MAP_KEYBOARD", "BTN_B"}, "KEY_BACK")
	output.SetPath([]string{"MAP_KEYBOARD", "BTN_SELECT"}, "KEY_COMPOSE")
	output.SetPath([]string{"MAP_KEYBOARD", "BTN_THUMBL"}, "KEY_HOME")

	LT_RT_BTN := false
	HAT0X, HAT0X_ok := abs[16]
	HAT0Y, HAT0Y_ok := abs[17]
	if HAT0X_ok && HAT0Y_ok {
		output.SetPath([]string{"ABS", "16"}, create_abs_rec("HAT0X", HAT0X.Min, HAT0X.Max))
		output.SetPath([]string{"ABS", "17"}, create_abs_rec("HAT0Y", HAT0Y.Min, HAT0Y.Max))
		if len(abs) == 6 { //四个轴+DPAD两个 则需要LT_RT_按键
			LT_RT_BTN = true
		}
	} else if keys[0x220] && keys[0x221] && keys[0x222] && keys[0x223] {
		output.SetPath([]string{"BTN", "544"}, "BTN_DPAD_UP")
		output.SetPath([]string{"BTN", "545"}, "BTN_DPAD_DOWN")
		output.SetPath([]string{"BTN", "546"}, "BTN_DPAD_LEFT")
		output.SetPath([]string{"BTN", "547"}, "BTN_DPAD_RIGHT")
		if len(abs) == 4 { //四个轴 则需要LT_RT_按键
			LT_RT_BTN = true
		}
	} else {
		logger.Warnf("未知DPAD种类 : %s", dev_name)
		return
	}

	need_keys := []string{"BTN_A", "BTN_B", "BTN_X", "BTN_Y", "BTN_LS", "BTN_RS", "BTN_LB", "BTN_RB", "BTN_SELECT", "BTN_START", "BTN_HOME"}
	if LT_RT_BTN {
		need_keys = append(need_keys, "BTN_LT", "BTN_RT")
	}

	for _, key_name := range need_keys {
		logger.Infof("press %s", key_name)
		output.SetPath([]string{"BTN", fmt.Sprintf("%d", get_key(pack_ch))}, key_name)
	}
	abs_map := get_abs_map(abs, pack_ch, LT_RT_BTN)
	for k, v := range abs_map {
		output.SetPath([]string{"ABS", fmt.Sprintf("%d", k)}, create_abs_rec(v, abs[evdev.AbsoluteType(k)].Min, abs[evdev.AbsoluteType(k)].Max))
	}

	jsonString, err := output.EncodePretty()
	if err != nil {
		logger.Errorf("%s\n", err)
	}
	logger.Infof("%s\n", jsonString)

	path, _ := exec.LookPath(os.Args[0])
	abspath, _ := filepath.Abs(path)
	workingDir, _ := filepath.Split(abspath)
	joystickInfosDir := filepath.Join(workingDir, "joystickInfos")
	if _, err := os.Stat(joystickInfosDir); os.IsNotExist(err) {
		os.Mkdir(joystickInfosDir, os.ModePerm)
	}
	savePath := filepath.Join(joystickInfosDir, fmt.Sprintf("%s.json", dev_name))
	logger.Infof("save to %s\n", savePath)
	err = ioutil.WriteFile(savePath, jsonString, 0644)
	if err != nil {
		logger.Errorf("%s\n", err)
	}
	return
}
