# 1. RTIO（Real Time Input Output Service for IoT）

> 简体中文 | [English](./README.md)

- [1. RTIO（Real Time Input Output Service for IoT）](#1-rtioreal-time-input-output-service-for-iot)
  - [1.1. 目标](#11-目标)
  - [1.2. 原理](#12-原理)
  - [1.3. 特点](#13-特点)
  - [1.4. 限制](#14-限制)
  - [1.5. 与MQTT比较](#15-与mqtt比较)
  - [1.6. 快速开始](#16-快速开始)
    - [1.6.1. 编译和运行](#161-编译和运行)
    - [1.6.2. 设备端代码](#162-设备端代码)
  - [1.7. 设备SDK](#17-设备sdk)
    - [1.7.1. Golang](#171-golang)
    - [1.7.2. C](#172-c)
  - [1.8. 更多](#18-更多)

---

## 1.1. 目标

通过几行关键代码，即可将“设备能力”以URI形式映射到云端或边缘端，可直接使用HTTP对设备访问、控制。

## 1.2. 原理

```text
                                                                 
   Device:PRINTER-001           Native │ Could or Edge                 
  ┌───────────────────────┐            │             ┌────────────┐ 
  │                       │            │             │            │ 
  │ /printer/action       │            │             │            │ 
  │ /printer/status       │  tcp/tls   │             │            │ 
  │                       │ ───────────┼───────────► │            │ 
  │                       │            │             │            │ 
  └───────────────────────┘            │             │            │ 
                                       │             │            │ 
   HTTPClient                          │             │            │ 
  ┌───────────────────────┐            │             │            │ 
  │ post $RTIO/PRINTER-001│            │             │            │ 
  │ req:                  │            │             │    rtio    │ 
  │ {                     │            │             │            │ 
  │   uri: /printer/action│            │             │            │ 
  │   data: cmd=start     │            │             │            │ 
  │ }                     │  http/https│             │            │ 
  │                       │ ───────────┼───────────► │            │ 
  │                       │ ◄─ ─ ─ ─ ─ ┼ ─ ─ ─ ─ ─ ─ │            │ 
  │ resp:                 │            │             │            │ 
  │ {                     │            │             │            │ 
  │   code: ok            │            │             │            │ 
  │   data: starting      │            │             │            │ 
  │ }                     │            │             │            │ 
  └───────────────────────┘            │             └────────────┘ 
   Phone/WEB/PC...                    

```

以上为示意图。

- 设备`PRINTER-001`与RTIO服务端建立起长连接。
- HTTPClient向设备`PRINTER-001`的URI`/printer/action`发出请求。
- RTIO将该请求转发到设备上并将设备处理结果通过HTTP响应传回给HTTPClient。

从调用端看，“设备能力”通过RTIO服务被映射到了云端，而不必关心其实现、通信等细节。

## 1.3. 特点

设备调用端不限于移动端、PC端或后台服务，以下统一称“调用端”

- 调用端直接使用HTTP调用设备，时限内返回设备处理结果，否则报错超时。
- 调用端通过HTTP可实时观察设备（非轮询），设备端通常是一个处理过程，比如不断返回处理进度。
- 设备端可通过`CoPOST`（类似`HTTP-POST`请求）向云端URI发起请求。
- 设备端URI在传输时采用`HASH摘要`（仅4字节），减少频繁调用传输URI对通信带宽浪费。
- HTTP访问支持JWT验证。
- 支持TLS和HTTPS。
- RTIO单节点支持百万连接，[旧版压测报告](https://github.com/guowenhe/rtio/blob/master/docs/stress_report.md)。
- RTIO单个执行文件（Golang实现），易于布署于云端或边缘端。
  
## 1.4. 限制

- 当前版本主要面向“控制指令”通信，一次调用中，请求载荷和（payload）、响应载荷最大长度都为504字节，具体参考[设备接入文档](./docs/device_access_protocol.md)。

## 1.5. 与MQTT比较

MQTT为发布订阅模型，更适合多对多通信。

RTIO为点对点通信模型，更适合比如手机控制设备等远程控制场景。以下为具体比较。

|项目            | MQTT       |RTIO    |
|:--------------|:------------|:-----------|
|通信模型        | 发布订阅模型 |点对点，REST-Like模型$^{1}$，支持观察者模式$^{2}$|
|调用端集成SDK   | 需要        |不需要，使用HTTP协议 |
|远程控制实现难度 | 高         |低$^{3}$  |
|协议轻量高效    | 是          |是$^{4}$ |
|双向通信        | 是          |是          |
|百万连接        | 支持        |支持        |
|可靠消息传输    | 支持        |支持$^{5}$|
|不可靠网络      | 支持        |支持$^{6}$|
|安全通信（TLS）  | 支持       |支持        |
|JWT验证       | -          |支持$^{7}$  |

**备注：**

1. REST-Like这里指类似RESTful模型（互联网广泛使用），但不以HTTP协议作为底层协议，提供了`CoPOST`（Constrained-Post，类似HTTP-POST）方法和`ObGET`（Observe-GET，观察者模式）方法，通过`URI`标识资源或能力。
2. 观察者模式类似MQTT消息订阅，但更为灵活，可以随时观察和取消。调用端仍然使用HTTP协议，无需额外操作（比如`订阅`）。
3. RTIO点对点模型更简单，相对MQTT无需`订阅`、`发布`等流程。远程控制设备等点对点通信场景下，比MQTT实现交互次数减少一半以上,且无Topic耦合（MQTT通常需要定义一对Topic表示该接口的请求和响应:`*_req`、`*_resp`）。参考：[FAQ](./docs/cn/rtio_faq.md)。
4. 支持受限设备，通常能运行于几十KB至百KB RAM的设备。RTIO每次通信不必传输URI（类似MQTT里的topic），通信效率更高。
5. RTIO没有QOS，可靠性依赖TCP，当时限内无法传达消息会直接向调用者报错。
6. 集成RTIO-SDK的设备端在网络断开后会重连。
7. HTTP调用端支持JWT验证，带有合法的JWT的请求才能请求RTIO服务，对设备发起调用。  

参考：

- [MQTT官网](https://mqtt.org/)

## 1.6. 快速开始

### 1.6.1. 编译和运行

环境需要安装golang（1.21版本以上）和make工具。

```sh
git clone https://github.com/mkrainbow/rtio.git
cd rtio
make
```

成功编译后，`out`目录下会生成执行文件，主要文件如下。

```sh
$ tree out/
out/
├── examples
│   ├── certificates
│   ├── simple_device
│   └── simple_device_tls
├── rtio
```

运行RTIO服务端，可通过`-h`参数查看帮助。

```sh
$ ./out/rtio -disable.deviceverify -disable.hubconfiger -log.level=info
INF cmd/rtio/rtio.go:83 > rtio starting ...
```

另一终端模拟设备。显示如下日志为成功连接到RTIO服务。

```sh
$ ./out/examples/simple_device
INF internal/devicehub/client/devicesession/devicesession.go:660 > serving device_ip=127.0.0.1:47032 deviceid=cfa09baa-4913-4ad7-a936-3e26f9671b09
INF internal/devicehub/client/devicesession/devicesession.go:601 > verify pass
```

再打开一个终端，通过`curl`访问RTIO服务，请求到设备的URI`/rainbow`，将"hello"字符发给设备，设备响应"world"。

```sh
$ curl http://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b09 -d '{"method":"copost", "uri":"/rainbow","id":12667,"data":"aGVsbG8="}'
{"id":12667,"fid":0,"code":"OK","data":"d29ybGQ="}

```

其中，"aGVsbG8="为"hello"的base64编码，"d29ybGQ="为"world"的base64编码。可通过以下命令在终端里编解码。

```sh

$ echo -n "hello" | base64       # Encode
aGVsbG8=
$ echo -n "d29ybGQ=" | base64 -d # Decode
world
```

### 1.6.2. 设备端代码

下面是Golang设备端实现。

```go
import (
    "github.com/mkrainbow/rtio-device-sdk-go/rtio"
)

func main() {

    // Connect to rtio service.
    session, err := rtio.Connect(context.Background(), *deviceID, *deviceSecret, *serverAddr)

    // ...
    
    // Register handler for URI.
    session.RegisterCoPostHandler("/rainbow", func(req []byte) ([]byte, error) {
        log.Printf("received [%s] and reply [world]", string(req))
        return []byte("world"), nil
    })

    // Session serve in the background.
    session.Serve(context.Background())

    // Do other things.
    time.Sleep(time.Hour * 8760)
}
```

## 1.7. 设备SDK

### 1.7.1. Golang

RTIO设备端SDK，Golang版:  

- rtio-device-sdk-go - <https://github.com/mkrainbow/rtio-device-sdk-go>

通常适合运行完整Linux的设备，比如ARM32位及以上单板。

### 1.7.2. C

RTIO设备端SDK，C语言版:  

- rtio-device-sdk-c - <https://github.com/mkrainbow/rtio-device-sdk-c>

适合资源受限设备，比如运行在FreeRTOS等实时操作系统上。

## 1.8. 更多

- [更多Demo](./docs/cn/rtio_demos.md)
- [设备接入协议](./docs/cn/device_access_protocol.md)
- [HTTP API](./docs/cn/http_access_protocol.md)
- [FAQ](./docs/cn/rtio_faq.md)
