# 1. RTIO Demos

> 简体中文 | [English](../rtio_demos.md)

- [1. RTIO Demos](#1-rtio-demos)
  - [1.1. 设备端通过TLS连接RTIO服务](#11-设备端通过tls连接rtio服务)
  - [1.2. 对设备进行观察](#12-对设备进行观察)
  - [1.3. 设备向透过RTIO向“设备服务”请求](#13-设备向透过rtio向设备服务请求)
    - [1.3.1. 启动RTIO配置服务](#131-启动rtio配置服务)
    - [1.3.2. 启动RTIO服务](#132-启动rtio服务)
    - [1.3.3. 访问设备服务](#133-访问设备服务)
  - [1.4. RTIO开启设备认证](#14-rtio开启设备认证)
    - [1.4.1. 启用设备认证服务](#141-启用设备认证服务)
    - [1.4.2. 启动RTIO服务](#142-启动rtio服务)
    - [1.4.3. 运行设备](#143-运行设备)
  - [1.5. RTIO开启HTTPS和JWT验证](#15-rtio开启https和jwt验证)
    - [1.5.1. 获取访问设备的JWT](#151-获取访问设备的jwt)
    - [1.5.2. 启动RTIO服务](#152-启动rtio服务)
    - [1.5.3. 运行设备](#153-运行设备)
    - [1.5.4. HTTP客户端访问](#154-http客户端访问)

## 1.1. 设备端通过TLS连接RTIO服务

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

运行RTIO过`-h`参数查看帮助。

```sh
$ ./out/rtio -disable.deviceverify  \
             -disable.hubconfiger \
             -log.level=debug \
             -enable.hub.tls \
             -tls.hub.certfile=./out/examples/certificates/demo_server.crt \
             -tls.hub.keyfile=./out/examples/certificates/demo_server.key

INF cmd/rtio/rtio.go:83 > rtio starting ...
INF internal/devicehub/server/devicetcp/devicetls.go:123 > TLS access enabled
```

另一终端模拟设备。显示如下日志为成功连接到RTIO服务。

```sh
$ ./out/examples/simple_device_tls
INF internal/devicehub/client/devicesession/devicesession.go:660 > serving device_ip=127.0.0.1:47032 deviceid=cfa09baa-4913-4ad7-a936-3e26f9671b09
INF internal/devicehub/client/devicesession/devicesession.go:601 > verify pass
```

再打开一个终端，`curl`透过RTIO备的URI`/rainbow`，将"hello"字符发给设备，设备响应"world"。

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

## 1.2. 对设备进行观察

成功编译后，`out`目录下会生成执行文件，主要文件如下。

```sh
$ tree out/
out/
├── examples
│   ├── certificates
│   ├── simple_device
│   └── simple_device_obget
├── rtio
```

运行RTIO过`-h`参数查看帮助。

```sh
$ ./out/rtio -disable.deviceverify -disable.hubconfiger -log.level=debug

INF cmd/rtio/rtio.go:83 > rtio starting ...
INF internal/devicehub/server/devicetcp/devicetls.go:123 > TLS access enabled
```

另一终端模拟设备。显示如下日志为成功连接到RTIO服务。

```sh
$ ./out/examples/simple_device_obget
INF internal/devicehub/client/devicesession/devicesession.go:660 > serving device_ip=127.0.0.1:47032 deviceid=cfa09baa-4913-4ad7-a936-3e26f9671b09
INF internal/devicehub/client/devicesession/devicesession.go:601 > verify pass
```

再打开一个终端，`curl`透过RTIO备的URI`/rainbow`，将"hello"字符发给设备，设备响应"world0"至"world9"。

```sh
$ curl http://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b09 -d '{"method":"obget", "uri":"/rainbow","id":12667,"data":"aGVsbG8="}'
{"id":12667,"fid":0,"code":"CONTINUE","data":"d29ybGQhIDA="}
{"id":12667,"fid":1,"code":"CONTINUE","data":"d29ybGQhIDE="}
{"id":12667,"fid":2,"code":"CONTINUE","data":"d29ybGQhIDI="}
{"id":12667,"fid":3,"code":"CONTINUE","data":"d29ybGQhIDM="}
{"id":12667,"fid":4,"code":"CONTINUE","data":"d29ybGQhIDQ="}
{"id":12667,"fid":5,"code":"CONTINUE","data":"d29ybGQhIDU="}
{"id":12667,"fid":6,"code":"CONTINUE","data":"d29ybGQhIDY="}
{"id":12667,"fid":7,"code":"CONTINUE","data":"d29ybGQhIDc="}
{"id":12667,"fid":8,"code":"CONTINUE","data":"d29ybGQhIDg="}
{"id":12667,"fid":9,"code":"CONTINUE","data":"d29ybGQhIDk="}
{"id":12667,"fid":10,"code":"TERMINATE","data":""}
```

可通过以下命令在终端里对base64的`data`编解码。

```sh
$ echo -n "hello" | base64           # Encode
aGVsbG8=
$ echo -n "d29ybGQhIDA=" | base64 -d # Decode
world! 0
$ echo -n "d29ybGQhIDE=" | base64 -d # Decode
world! 1

# ...

$ echo -n "d29ybGQhIDk=" | base64 -d # Decode
world! 9
```

## 1.3. 设备向透过RTIO向“设备服务”请求

设备端通过CoPOST方式请求RTIO所代理的后端服务，这里称为设备服务。该服务的动态更新通过配置服务`hubconfiger`来实现。

配置服务实现可参考[配置服务接口](./http_hubconfiger.md)，[配置服务示例代码](../examples/hubconfiger/hubconfiger.go)。

### 1.3.1. 启动RTIO配置服务

```sh
$ ./out/examples/hubconfiger
INF hubconfiger.go:150 > hubconfiger started httpaddr=0.0.0.0:17317
INF hubconfiger.go:151 > hubconfiger URL, replace the 0.0.0.0 address with the external address.  url=http://0.0.0.0:17317/hubconfiger
```

### 1.3.2. 启动RTIO服务

日志里显示了所加载的设备服务。

```sh
$ ./out/rtio -disable.deviceverify -backend.hubconfiger=http://localhost:17317/hubconfiger -log.level=debug
DBG cmd/rtio/rtio.go:71 > Config:backend.hubconfiger=http://localhost:17317/hubconfiger
INF cmd/rtio/rtio.go:81 > rtio starting ...
# ...
DBG configer.go:156 > Config: backend.hubconfiger=http://localhost:17317/hubconfiger
DBG configer.go:156 > Config: deviceservice.a796d057=http://localhost:17517/deviceservice/aa/bb
DBG configer.go:156 > Config: deviceservice.c98ad180=http://localhost:17517/deviceservice/aa/cc
DBG configer.go:156 > Config: deviceservice.18afd2e4=http://localhost:17518/deviceservice/aa/dd
```

### 1.3.3. 访问设备服务

设备每5秒钟请求URI"/aa/bb"服务，参考[示例代码](../examples/simple_device_copost_to_server/simple_device_copost_to_server.go)。

```sh
$ ./out/examples/simple_device_copost_to_server
2024-09-16 13:02:33.266 INFdevicesession.go:772 > serving device_ip=127.0.0.1:41548 deviceid=cfa09baa-4913-4ad7-a936-3e26f9671b09

# after 5 seconds
2024/09/16 13:02:38 resp=deviceservice: respone with bb
```

设备服务日志，如下。

```sh
INF deviceservice.go:113 > req data data="test for device post" deviceid=cfa09baa-4913-4ad7-a936-3e26f9671b09 id=182274021
INF deviceservice.go:117 > resp data data="deviceservice: respone with bb" deviceid=cfa09baa-4913-4ad7-a936-3e26f9671b09 id=182274021
```

## 1.4. RTIO开启设备认证

### 1.4.1. 启用设备认证服务

以下为认证服务Demo，可参考[设备认证服务接口](./http_deviceverifier.md)实现认证服务。

```sh
$ ./out/examples/deviceverifier -h
Usage of ./out/examples/deviceverifier:
  -http.addr string
        address for http connection (default "0.0.0.0:17217")
$ ./out/examples/deviceverifier
INF deviceverifier.go:127 > deviceverifier started httpaddr=0.0.0.0:17217
INF deviceverifier.go:129 > deviceverifier URL, replace the 0.0.0.0 address with the external address.  url=http://0.0.0.0:17217/deviceverifier
```

### 1.4.2. 启动RTIO服务

移除`-disable.deviceverify`选项，同时增加`-backend.deviceverifier=localhost:17217`选项。

```sh
$ ./out/rtio -backend.deviceverifier=http://localhost:17217/deviceverifier -disable.hubconfiger -log.level=debug
DBG cmd/rtio/rtio.go:73 > Config:backend.deviceverifier=http://localhost:17217/deviceverifier
# ...
INF cmd/rtio/rtio.go:83 > rtio starting ...
```

### 1.4.3. 运行设备

默认的`DeviceID`和`DeviceSecret`已经设定在认证服务中。[参考代码](../examples/deviceverifier/deviceverifier.go)。

```sh
$ ./out/examples/simple_device -id="cfa09baa-4913-4ad7-a936-3e26f9671b09" -secret="mb6bgso4EChvyzA05thF9+wH"
INF devicesession.go:772 > serving device_ip=127.0.0.1:33456 deviceid=cfa09baa-4913-4ad7-a936-3e26f9671b09
# ...
INF devicesession.go:713 > verify pass
```

使用错误的`DeviceSecret`时设备无法服务连接到RTIO服务。

```sh
$ ./out/examples/simple_device -id="cfa09baa-4913-4ad7-a936-3e26f9671b09" -secret="mb6bgso4EChvyzA05thF9+xx"
INF devicesession.go:772 > serving device_ip=127.0.0.1:39646 deviceid=cfa09baa-4913-4ad7-a936-3e26f9671b09
ERR devicesession.go:715 > verify fail resp.code=3
ERR devicesession.go:805 > device done when error error=ErrVerifyFailed
```

## 1.5. RTIO开启HTTPS和JWT验证

### 1.5.1. 获取访问设备的JWT

可参考: [JWT签发示例服务](./http_jwtissuer.md)， 比如以下获取到JWT。

```sh
$ curl http://localhost:17019/jwtissuer -d '{"method":"jwtissuer", "id":12667,"deviceid":"cfa09baa-4913-4ad7-a936-3e26f9671b09", "expires": 86400}'
{"id":12667,"code":"OK","jwt":"eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjY0NjQ1MzAsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0zZTI2Zjk2NzFiMDkifQ.sM80Zu3nMsmahzmLpmSC4GlsI8G0xKbnIk8kZIJvLH9IbadEc3sOM3tSb7m_L_ZY2eWy4Ipl8EiYS7t9y_NmCA"}
```

### 1.5.2. 启动RTIO服务

```sh
$ ./out/rtio \
-disable.deviceverify  \
-disable.hubconfiger \
-log.level=debug \
-enable.hub.tls \
-tls.hub.certfile=./out/examples/certificates/demo_server.crt \
-tls.hub.keyfile=./out/examples/certificates/demo_server.key \
-enable.https \
-https.certfile=./out/examples/certificates/demo_server.crt \
-https.keyfile=./out/examples/certificates/demo_server.key \
-enable.jwt \
-jwt.ed25519=./out/examples/certificates/ed25519.public

INF cmd/rtio/rtio.go:83 > rtio starting ...
INF internal/devicehub/server/devicetcp/devicetls.go:123 > TLS access enabled
INF internal/httpaccess/server/httpgw/httpgw.go:464 > gateway started with TLS gwaddr=0.0.0.0:17917
```

### 1.5.3. 运行设备

```sh
./out/examples/simple_device_tls -with-ca ./out/examples/certificates/ca.crt
```

### 1.5.4. HTTP客户端访问

先将JWT赋值给变量`TOKEN`,然后通过`-H`指定在请求Header中传入JWT。

```sh
$ export TOKEN="eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjY0NjQ1MzAsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0zZTI2Zjk2NzFiMDkifQ.sM80Zu3nMsmahzmLpmSC4GlsI8G0xKbnIk8kZIJvLH9IbadEc3sOM3tSb7m_L_ZY2eWy4Ipl8EiYS7t9y_NmCA"
$ curl https://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b09 -H "Authorization:Bearer $TOKEN" -d '{"method":"copost", "uri":"/rainbow","id":12667,"data":"aGVsbG8="}' --cacert ./out/examples/certificates/ca.crt
{"id":12667,"fid":0,"code":"OK","data":"d29ybGQ="}
```

详细日志如下：

```sh
$ curl https://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b09 -H "Authorization:Bearer $TOKEN" -d '{"method":"copost", "uri":"/rainbow","id":12667,"data":"aGVsbG8="}' --cacert ./out/examples/certificates/ca.crt  -v
> POST /cfa09baa-4913-4ad7-a936-3e26f9671b09 HTTP/2
> Host: localhost:17917
> user-agent: curl/7.81.0
> accept: */*
> authorization:Bearer eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjY0NjQ1MzAsImlzcyI6InJ0aW8iLCJzdWIiOiJjZmEwOWJhYS00OTEzLTRhZDctYTkzNi0zZTI2Zjk2NzFiMDkifQ.sM80Zu3nMsmahzmLpmSC4GlsI8G0xKbnIk8kZIJvLH9IbadEc3sOM3tSb7m_L_ZY2eWy4Ipl8EiYS7t9y_NmCA
> content-length: 66
> content-type: application/x-www-form-urlencoded
>

< HTTP/2 200
< access-control-allow-headers: Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization
< access-control-allow-methods: POST, GET, OPTIONS, PUT, DELETE
< access-control-allow-origin: *
< content-type: text/plain; charset=utf-8
< content-length: 50
< date: Sun, 15 Sep 2024 06:10:07 GMT
<
{"id":12667,"fid":0,"code":"OK","data":"d29ybGQ="}
```
