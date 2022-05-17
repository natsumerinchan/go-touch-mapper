# 安卓触屏映射工具

将手柄,键鼠操作映射为触屏事件

## Run

从[releases](https://github.com/DriverLin/go-touch-mapper/releases)下载最新版本的 go-touch-mapper 二进制文件

使用adb执行(无需root)
```
adb push go-touch-mapper /data/local/tmp
adb shell 
cd /data/local/tmp
chmod 777 ./go-touch-mapper
./go-touch-mapper -h
```

root下直接运行
```
wget wget https://github.com/DriverLin/go-touch-mapper/releases/latest/download/go-touch-mapper
chmod 777 ./go-touch-mapper
sudo ./go-touch-mapper -h
```

## Build & Run

使用termux编译并运行
```
pkg i golang -y
git clone https://github.com/DriverLin/go-touch-mapper
cd go-touch-mapper
go build
sudo ./go-touch-mapper -h
```

## Help

```
usage: go-touch-mappeer [-h|--help] [-a|--auto-detect] [--create-js-info]
                        [-e|--event] [-t|--touch <integer>] [-c|--config
                        "<value>"] [-i|--inputManager] [-r|--remoteControl]
                        [-p|--port <integer>]



Arguments:

  -h  --help            Print help information
  -a  --auto-detect     自动检测设备. Default: false
      --create-js-info  创建手柄配置文件. Default: false
  -e  --event           键盘或鼠标或手柄的设备号
  -t  --touch
                        触屏设备号,可选,当指定时可同时使用映射与触屏而不冲突.
                        Default: -1
  -c  --config          配置文件路径
  -i  --inputManager    是否使用inputManager,需开启额外控制进程.
                        Default: false
  -r  --remoteControl   是否从UDP接收远程事件. Default: false
  -p  --port            指定监听远程事件的UDP端口号,默认61069.
                        Default: 61069
```
## Example

```
touchScreen /dev/input/event5
mouse /dev/input/event15
keyboard /dev/input/event19
joystick /dev/input/event16
configFile SMC_JS.json 
```

手动指定每个设备号
```
go-touch-mapper -t 5 -e 15 -e 19 -e 16 -c configs/SMC_JS.json  
```
或者使用自动检测
```
go-touch-mapper -a -c configs/SMC_JS.json
```



## Switch Mode
默认键盘使用 ` 键，可在配置文件中修改，手柄为SELECT+RS组合键

## ConfigFile
[创建映射文件工具](https://driverlin.github.io/go-touch-mapper/build/)

如有需要，可参照[EXAMPLE.JSON](https://github.com/DriverLin/go-touch-mapper/blob/main/configs/EXAMPLE.JSON)手动修改配置文件

左上角为原点，向右为X，向下为Y

## Joystick
joystickInfos文件夹下json文件为手柄配置文件

按照按键位置统一为XBOX手柄按键

扳机键按照力度映射为5个等级的按键 BTN_LT_1 ... BTN_LT_5

没有线性扳机的手柄例如 Switch Pro controller 只有 BTN_LT 和 BTN_RT

### 手动创建手柄配置文件
```
go-touch-mapper --create-js-info -e 16 
```
或使用自动检测
```
go-touch-mapper --create-js-info -a
```

## RemoteControl
```
go-touch-mapper -t 5 -c configs/SMC_JS.json -r
```
PC上 修改remote_control中sender("192.168.1.64:61069")为手机IP，端口默认可不修改

运行

``` 
python remote_control.py 
```

## [InputManager](https://github.com/DriverLin/inputManager-touch-interface)
### 说明
由Scrcpy修改而来，在ADB下使用app_process启动

通过UDP与主程序通信控制触屏

可在多显示器情况下指定displayID

### 使用

```
adb push inputManager/app-debug.apk /data/local/tmp/
adb shell CLASSPATH=/data/local/tmp/app-debug.apk  app_process ./ com.genymobile.scrcpy.Server 0
```

```
go-touch-mapper -t 5 -e 15 -e 19 -e 16 -c configs/SMC_JS.json -i
```

