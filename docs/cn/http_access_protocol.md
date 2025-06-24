# http access protocol

> 简体中文 | [English](../http_access_protocol.md)

## 接口定义

接口使用HTTP通信，如果RTIO开启JWT验证，则需要HTTP请求的Header中加入"Authorization"字段。

URL构成，HOST为RTIO服务主机地址，DEVICE_ID为设备标识。

```text
http://$HOST/$DEVICE_ID
```

请求参数，编码为JSON字符串，目前版本限定字符串总长度为864¹字节。

|参数 |类型   |长度|必选 | 描述|
|:---|:------|:-------|:---|:-----|
| method|string | 1-10  |是|请求设备的方法，仅支持copost和obget|
| id |uint32 |-   |是|请求标识，每个请求唯一，响应中该字段会与之匹配|
| uri|string |3-128  |是|设备内部的uri，会绑定handler到该uri上|
| data |base64 | 0-672² |否|为base64字符串|

响应参数，编码为JSON字符串。

|参数 |类型   |长度|必选 | 描述|
|:---|:------|:-------|:---|:-----|
| code|string | 0-64 |是|错误码|
| id |uint32 | - |是|应答标识，与请求中该字段匹配|
| fid |uint32 | - |否|obget方法必传，Frame标识，用于标识每个Frame|
| data |base64 | 0-672² |否|为base64字符串|

备注：

1. RTIO服务收到请求会检查JSON字符串总长度，为以上数据长度与JSON引号、括号、字段名等数据长度之和，这里设定为864字节。
2. 设备与RTIO服务建立连接时（可参考设备接入协议），约定一次传输最大Body长度，目前仅支持512字节。目前REST-Like通信层预留8字节，一次通信最大长度为504字节,再将其编码为base64，长度为672字节（504*4/3）。

## 样例

通过`copost`方法请求设备（设备在线）：

```sh
$ curl https://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b10 -d '{"method":"copost", "uri":"/greeter","id":12667,"data":"c3RhcnQ="}' 

{"id":12667, "code":"OK", "data":"d29ybGQ="}
```

通过`copost`方法请求设备（设备离线）：

```sh
$ curl https://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b09 -d '{"method":"copost", "uri":"/greeter","id":12667,"data":"aGVsbG8="}'
{"id":12667,"fid":0,"code":"DEVICEID_OFFLINE","data":""}
```

通过`copost`方法请求设备（设备超时）：

```sh
$ curl https://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b10 -d '{"method":"copost", "uri":"/greeter","id":12667,"data":"c3RhcnQ="}' 

{"id":12667, "code":"DEVICEID_TIMEOUT"}
```

通过`obget`请求设备：

```sh
curl https://localhost:17917/cfa09baa-4913-4ad7-a936-3e26f9671b10 -d '{"method":"obget", "uri":"/greeter","id":12667,"data":"c3RhcnQ="}' 

{"id":12334,"fid":0,"code":"CONTINUE","data":"cHJpbnRpbmcgMzMl"}
{"id":12334,"fid":1,"code":"CONTINUE","data":"cHJpbnRpbmcgMzcl"}
...
{"id":12334,"fid":21,"code":"TERMINATE","data":""}
```

## 更多样例

参考：[RTIO Demos](./rtio_demos.md)

## 错误码

以下为RTIO错误码，HTTP作为RTIO的传输层，通常HTTP响应码为200时才能正确返回JSON数据。

|code | 定义                          | 描述      |
|:----|:-----------------------------|:---------|
| 0   | INTERNAL_SERVER_ERROR  | 内部错误  |
| 1   | OK                     | 成功      |
| 2   | DEVICEID_OFFLINE       | 设备已下线 |
| 3   | DEVICEID_TIMEOUT       | 设备超时，观察者接口不会请求超时，但由于网络问题设备会没响应 |
| 4   | CONTINUE               | 观察模式下，存在后续数据帧 |
| 5   | TERMINATE              | 观察模式下，不存在后续数据帧 |
| 6   | NOT_FOUND              | URI找不到 |
| 7   | BAD_REQUEST            | 请求数据无效 |
| 8   | METHOD_NOT_ALLOWED     | 请求设备方法错误，当前仅支持copost和obget |
| 9   | TOO_MANY_REQUESTS      | 太多请求 |
| 10  | TOO_MANY_OBSERVERS     | 太多观察者 |
| 11  | REQUEST_TIMEOUT        | 请求超时 |
