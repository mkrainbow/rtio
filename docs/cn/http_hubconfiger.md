# DeviceHub配置服务

> 简体中文 | [English](../http_hubconfiger.md)

## 配置服务接口

动态更新RTIO配置，目前仅限设备服务地址更新。使用HTTP通信，POST方法发送。

URL示例：

```text
http://$HOST/hubconfiger
```

请求参数，编码为JSON字符串。

|参数 |类型   |长度|必选 | 描述|
|:---|:------|:-------|:---|:-----|
| method|string | 1-10  |是|这里`method`为`getconfig`|
| id |uint32 |-   |是|请求标识，每个请求唯一，响应中该字段会与之匹配|

响应参数，编码为JSON字符串。

|参数 |类型   |长度|必选 | 描述|
|:---|:------|:-------|:---|:-----|
| code|string | 0-128 |是|错误码|
| id |uint32 | - |是|应答标识，与请求中该字段匹配|
| config |string | 0-2048 |是|为JSON编码的String|
| digest |uint32 | - |是|`config` 字符串的摘要，hash函数为CRC32|

## 错误码

以下为RTIO错误码，HTTP作为RTIO的传输层，通常HTTP响应码为200时才能正确返回JSON数据。

| code                   | 描述      |
|:-----------------------|:---------|
| INTERNAL_SERVER_ERROR  | 内部错误  |
| OK                     | 成功      |
| BAD_REQUEST            | 请求数据无效 |
| METHOD_NOT_ALLOWED     | 请求设备方法错误 |

## 示例

```sh
$ curl http://localhost:17317/hubconfiger -d '{"method":"getconfig","id": 1999 }'
{"id":1999,"code":"OK","config":"{\"deviceservicemap\":{\"/aa/bb\":\"http://localhost:17517/deviceservice/aa/bb\",\"/aa/cc\":\"http://localhost:17517/deviceservice/aa/cc\",\"/aa/dd\":\"http://localhost:17517/deviceservice/aa/dd\"}}","digest":785914363}

```
