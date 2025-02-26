# RECORD插件

对流进行录制的功能插件，提供Flv、fmp4、hls、裸流格式的录制功能。

## 插件地址

https://github.com/Monibuca/plugin-record

## 插件引入
```go
import (
    _ "m7s.live/plugin/record/v4"
)
```
## 配置

- 配置中的path 表示要保存的文件的根路径，可以使用相对路径或者绝对路径
- filter 代表要过滤的StreamPath正则表达式，如果不匹配，则表示不录制。为空代表不进行过滤
- fragment表示分片大小（秒），0代表不分片

```yaml
record:
  subscribe: # 参考全局配置格式
  flv:
      ext: .flv
      path: record/flv
      autorecord: false
      filter: ""
      fragment: 0
  mp4:
      ext: .mp4
      path: record/mp4
      autorecord: false
      filter: ""
      fragment: 0
  hls:
      ext: .m3u8
      path: record/hls
      autorecord: false
      filter: ""
      fragment: 0
  raw:
      ext: .
      path: record/raw
      autorecord: false
      filter: ""
      fragment: 0
```

## API

- `/record/api/list/recording` 罗列所有正在录制中的流的信息
- `/record/api/list?type=[flv|mp4|hls|raw]` 罗列所有录制的flv|mp4|m3u8|raw文件
- `/record/api/start?type=flv&streamPath=live/rtc&fileName=xxx&fragment=10s` 开始录制某个流,返回一个字符串用于停止录制用的id(fileName是可选的，且只用于非切片情况,fragment用于覆盖配置中的切片时间，是可选的)
- `/record/api/stop?id=xxx` 停止录制某个流

## 点播功能

访问格式：
 [http/https]://[host]:[port]/record/[streamPath].[flv|mp4|m3u8|h264|h265]

例如：
- `http://localhost:8080/record/live/test.flv` 将会读取对应的flv文件
- `http://localhost:8080/record/live/test.mp4` 将会读取对应的fmp4文件

