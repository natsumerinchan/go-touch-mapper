# 安卓触屏映射工具

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
usage: go-touch-mappeer [-h|--help] [-e|--event] [-t|--touch <integer>]
                        -c|--config "<value>" [-i|--inputManager]
                        [-r|--remoteControl] [-p|--port <integer>]



Arguments:

  -h  --help           Print help information
  -e  --event          键盘或鼠标或手柄的设备号
  -t  --touch
                       触屏设备号,可选,当指定时可同时使用映射与触屏而不冲突.
                       Default: -1
  -c  --config         配置文件路径
  -i  --inputManager   是否使用inputManager,需开启额外控制进程.
                       Default: false
  -r  --remoteControl  是否从UDP接收远程事件. Default: false
  -p  --port           指定监听远程事件的UDP端口号,默认61069.
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
```
go-touch-mapper -t 5 -e 15 -e 19 -e 16 -c ./SMC_JS.json  
```

## Switch Mode
默认键盘使用 ` 键，可在配置文件中修改，手柄为SELECT+RS组合键

## ConfigFile
[创建映射文件工具](https://driverlin.github.io/go-touch-mapper/build/)

如有需要，可参照[EXAMPLE.JSON](https://github.com/DriverLin/go-touch-mapper/blob/main/EXAMPLE.JSON)手动修改配置文件

左上角为原点，向右为X，向下为Y

## 远程控制
```
go-touch-mapper -t 5 -e 15 -e 19 -e 16 -c ./SMC_JS.json -r
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
go-touch-mapper -t 5 -e 15 -e 19 -e 16 -c ./SMC_JS.json -i
```

