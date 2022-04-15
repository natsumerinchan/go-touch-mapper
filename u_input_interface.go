package main

/*
#include <jni.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <fcntl.h>
#include <linux/input.h>
#include <linux/uinput.h>
#include <time.h>


int createTouchDevice(int sw, int sh)
{
    struct uinput_user_dev uinp;
    int width = sw;
    int height = sh;
    int fd_touch = open("/dev/uinput", O_WRONLY | O_NDELAY);
    if (fd_touch <= 0)
    {
        printf("open /dev/uinput failed\n");
        return -1;
    }
    memset(&uinp, 0x00, sizeof(uinp));
    strncpy(uinp.name, "myTouch", strlen("myTouch"));
    uinp.id.version = 1;
    uinp.id.bustype = BUS_USB;

    uinp.absmin[ABS_MT_POSITION_X] = 0;
    uinp.absmax[ABS_MT_POSITION_X] = sw;
    uinp.absfuzz[ABS_MT_POSITION_X] = 0;
    uinp.absflat[ABS_MT_POSITION_X] = 0;

    uinp.absmin[ABS_MT_POSITION_Y] = 0;
    uinp.absmax[ABS_MT_POSITION_Y] = sh;
    uinp.absfuzz[ABS_MT_POSITION_Y] = 0;
    uinp.absflat[ABS_MT_POSITION_Y] = 0;

    uinp.absmin[ABS_MT_TOUCH_MAJOR] = 0;
    uinp.absmax[ABS_MT_TOUCH_MAJOR] = 255;
    uinp.absfuzz[ABS_MT_TOUCH_MAJOR] = 0;
    uinp.absflat[ABS_MT_TOUCH_MAJOR] = 0;

    uinp.absmin[ABS_MT_WIDTH_MAJOR] = 0;
    uinp.absmax[ABS_MT_WIDTH_MAJOR] = 0;
    uinp.absfuzz[ABS_MT_WIDTH_MAJOR] = 0;
    uinp.absflat[ABS_MT_WIDTH_MAJOR] = 0;

    uinp.absmin[ABS_MT_SLOT] = 0;
    uinp.absmax[ABS_MT_SLOT] = 9;
    uinp.absfuzz[ABS_MT_SLOT] = 0;
    uinp.absflat[ABS_MT_SLOT] = 0;

    uinp.absmin[ABS_MT_TRACKING_ID] = 0;
    uinp.absmax[ABS_MT_TRACKING_ID] = 65535;
    uinp.absfuzz[ABS_MT_TRACKING_ID] = 0;
    uinp.absflat[ABS_MT_TRACKING_ID] = 0;

    ioctl(fd_touch, UI_SET_EVBIT, EV_KEY);
    ioctl(fd_touch, UI_SET_KEYBIT, 0x014a);
    ioctl(fd_touch, UI_SET_KEYBIT, 0x003e);

    ioctl(fd_touch, UI_SET_EVBIT, EV_ABS);
    ioctl(fd_touch, UI_SET_ABSBIT, ABS_MT_SLOT);
    ioctl(fd_touch, UI_SET_ABSBIT, ABS_MT_TRACKING_ID);

    ioctl(fd_touch, UI_SET_ABSBIT, ABS_MT_TOUCH_MAJOR);
    ioctl(fd_touch, UI_SET_ABSBIT, ABS_MT_WIDTH_MAJOR);
    ioctl(fd_touch, UI_SET_ABSBIT, ABS_MT_POSITION_X);
    ioctl(fd_touch, UI_SET_ABSBIT, ABS_MT_POSITION_Y);
    ioctl(fd_touch, UI_SET_PROPBIT, INPUT_PROP_DIRECT);

    if (write(fd_touch, &uinp, sizeof(uinp)) != sizeof(uinp))
    {
        printf("error: write uinput_user_dev\n");
        close(fd_touch);
        fd_touch = -1;
        return fd_touch;
    }

    if (ioctl(fd_touch, UI_DEV_CREATE))
    {
        printf("error: ioctl UI_DEV_CREATE\n");
        close(fd_touch);
        fd_touch = -1;
        return fd_touch;
    }else{
        char sysfs_device_name[256];
        ioctl(fd_touch, UI_GET_SYSNAME(sizeof(sysfs_device_name)), sysfs_device_name);
        char inputX[256];
        strncpy(inputX,sysfs_device_name+5,strlen(sysfs_device_name)-5);
        return  atoi(inputX);
    }
}
*/
import "C"
import (
	"fmt"
	"os/exec"
)

func create_u_input_touch_screen(width int, height int) string {
	dev_index := C.createTouchDevice(1349, 3119)
	cmd := fmt.Sprintf("ls /sys/devices/virtual/input/input%d | egrep -o 'event([0-9]+)'", dev_index)
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		fmt.Printf("%s", err)
		return ""
	} else {
		return fmt.Sprintf("/dev/input/%s", string(out))
	}
}

func handel_u_input_interface(u_input chan *u_input_control_pack) {
	dev_path := create_u_input_touch_screen(1349, 3119)
	fmt.Printf("dev_path:%s\n", dev_path)
}

// // printf("ls /sys/devices/virtual/input/%s | egrep -o 'event([0-9]+)' \n", sysfs_device_name);
// sprintf(cmd, "ls /sys/devices/virtual/input/%s | egrep -o 'event([0-9]+)'", sysfs_device_name);
// char result[256];
// ExecuteCMD(cmd, result);
// // printf("%s\n", result);
// char devpath [64];
// sprintf(devpath, "/dev/input/%s", result);
// printf("%s\n", devpath);
