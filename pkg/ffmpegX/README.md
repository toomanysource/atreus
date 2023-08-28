# ffmpeg package 使用方法
## 1. 安装
在Linux下安装ffmpeg
```shell 
sudo apt-get install ffmpeg
```
## 2. 使用

```go
package main

import (
	"ffmpegX"
	"github.com/go-kratos/kratos/v2/log"
)

func main() {
	// 第一个参数为需要读取的文件路径
	// 第二个参数为读取的帧号
	// 返回值为io.Reader, error
	reader, err := ffmpegX.ReadFrameAsImage("./test1.mp4", 2)
	if err != nil {
		log.Fatal(err)
	}
	// 保存为图片
	// 第一个参数 io.Reader,第二个参数为保存的路径
	// 返回错误
	err = ffmpegX.SaveImage(reader, "./test1.jpg")
	if err != nil {
        log.Fatal(err)
    }
}
```
