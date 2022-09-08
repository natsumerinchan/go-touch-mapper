package main

import (
	"bufio"
	"encoding/binary"
	"net"
	"os"
)

func handel_touch_using_input_manager(control_ch chan *touch_control_pack) {
	unixAddr, err := net.ResolveUnixAddr("unix", "@unix_socket")
	if err != nil {
		logger.Errorf("创建Unix Domain Socket失败 : %s", err.Error())
		os.Exit(3)
	}
	unixListener, _ := net.ListenUnix("unix", unixAddr)
	defer unixListener.Close()
	logger.Info("waiting for input manager to connect")
	unixConn, _ := unixListener.AcceptUnix()
	defer unixConn.Close()
	logger.Info("input manager connected")
	writer := bufio.NewWriter(unixConn)

	for {
		select {
		case <-global_close_signal:
			return
		case control_data := <-control_ch:
			// logger.Debugf("控制设备 : %v", control_data)
			action := byte(control_data.action)
			id := byte(control_data.id & 0xff)
			x := make([]byte, 4)
			y := make([]byte, 4)
			binary.LittleEndian.PutUint32(x, uint32(control_data.x))
			binary.LittleEndian.PutUint32(y, uint32(control_data.y))
			// logger.Debugf("接收到控制命令 : %v", control_data)
			// logger.Debugf("write bytes : %v", []byte{action, id, x[0], x[1], x[2], x[3], y[0], y[1], y[2], y[3]})
			writer.Write([]byte{action, id, x[0], x[1], x[2], x[3], y[0], y[1], y[2], y[3]})
			writer.Flush()
		}
	}
}
