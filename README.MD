# 安卓触屏映射工具

将手柄,键鼠操作映射为触屏事件

## Quick start

从release直接下载编译好的二进制程序

一般手机平板下载 [go-touch-mapper_arm64](https://github.com/DriverLin/go-touch-mapper/releases/latest/download/go-touch-mapper_arm64)

androidX86下载 [go-touch-mapper_amd64](https://github.com/DriverLin/go-touch-mapper/releases/latest/download/go-touch-mapper_amd64)

读取dev和创建uinput需要shell或者root权限

所以可以用adb shell或者su shell运行

```
cd /data/local/tmp
curl -LJo go-touch-mapper https://github.com/DriverLin/go-touch-mapper/releases/latest/download/go-touch-mapper_arm64
chmod 777 ./go-touch-mapper
./go-touch-mapper -h
```

## Build & Run

环境 termux
```
pkg i golang -y
git clone https://github.com/DriverLin/go-touch-mapper
cd go-touch-mapper
CGO_ENABLE=0 GOOS=linux go build -ldflags="-s -w"
./go-touch-mapper -h
```

## Help

```
usage: go-touch-mappeer [-h|--help] [-a|--auto-detect] [--create-js-info]
                        [-e|--event] [-t|--touch <integer>] [-c|--config
                        "<value>"] [-i|--inputManager] [-r|--remoteControl]
                        [-p|--port <integer>] [-v|--v-mouse] [--auto-release
                        <integer>] [--measure-mode] [-d|--debug_mode]

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
  -p  --port            指定监听远程事件的UDP端口号. Default: 61069
  -v  --v-mouse
                        用触摸操作模拟鼠标,需要额光标外显示程序.
                        Default: false
      --auto-release
                        触发视角自动释放所需的静止ms数,50ms为检查单位,置0禁用.
                        Default: 200
      --measure-mode
                        显示视角移动像素计数,且可输入数值模拟滑动,方向键可微调.
                        Default: false
  -d  --debug_mode      打印debug信息. Default: false
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
大多数情况下，如果你使用了 -a 来自动检测，那么便无需输入 -e 与 -t ，程序会尝试自动检测对应的设备

如果自动检测漏掉了某些设备，你仍然可以使用 -e 与 -t 来手动添加它们

## 映射开关
默认键盘使用 ` 键，可在配置文件中修改，手柄为SELECT+RS组合键

## 配置文件
[创建映射文件工具](https://driverlin.github.io/go-touch-mapper/build/)

如有需要，可参照[EXAMPLE.JSON](https://github.com/DriverLin/go-touch-mapper/blob/main/configs/EXAMPLE.JSON)手动修改配置文件

左上角为原点，向右为X，向下为Y

按住键盘按键后，点击即可添加按键映射

## 手柄支持
joystickInfos文件夹下json文件为手柄配置文件

按照按键位置统一为XBOX手柄按键

扳机键按照力度映射为5个等级的按键 BTN_LT_1 ... BTN_LT_5

没有线性扳机的手柄例如 Switch Pro controller 只有 BTN_LT 和 BTN_RT

  注意⚠ : 

* 手柄的按键代码在不同设备上可能有所差异
* 如果出现按键无反应的情况，建议重新创建手柄配置文件

### 手动创建手柄配置文件
```
go-touch-mapper --create-js-info -e 16 
```
或使用自动检测
```
go-touch-mapper --create-js-info -a
```

## 远程控制
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
由Scrcpy修改而来，使用app_process启动

通过 Unix Domain Socket 与主程序通信控制触屏

可在多显示器情况下指定displayID

只能在adb shell下启用 

且由于Unix Domain Socket的相关权限设定，主程序也必须运行在adb shell下

### 使用

###### 先启动主程序(server)
```
go-touch-mapper -a -c configs/SMC_JS.json -i
```

###### 再启动InputManager(client)
```
adb push inputManager/app-debug.apk /data/local/tmp/
adb shell CLASSPATH=/data/local/tmp/app-debug.apk  app_process ./ com.genymobile.scrcpy.Server 0
```

