package main

var keycode_2_friendly_name map[uint16]string = map[uint16]string{}

func GetKeyName(keycode interface{}) string {
	switch keycode.(type) {
	case string:
		return keycode.(string)
	case uint16:
		if keycode_2_friendly_name[keycode.(uint16)] != "" {
			return keycode_2_friendly_name[keycode.(uint16)]
		}
		return ""
	default:
		return ""
	}
}
