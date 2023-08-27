package ffmpegX

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/disintegration/imaging"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

// ReadFrameAsImage 读取视频文件的某一帧并转换为jpeg格式
// inFilePath: 输入文件路径
// frameNum: 帧号
func ReadFrameAsImage(inFilePath string, frameNum int) (io.Reader, error) {
	buf := bytes.NewBuffer(nil)
	err := ffmpeg.Input(inFilePath).
		Filter("select", ffmpeg.Args{fmt.Sprintf("gte(n,%d)", frameNum)}).
		Output("pipe:", ffmpeg.KwArgs{"vframes": 1, "format": "image2", "vcodec": "png"}).
		WithOutput(buf, os.Stdout).
		Run()
	if err != nil {
		return nil, err
	}
	return buf, err
}

// SaveImage 保存jpeg格式的图片
// reader: 输入的图片流
// outFilePath: 输出文件路径
func SaveImage(reader io.Reader, outFilePath string) error {
	img, err := imaging.Decode(reader)
	if err != nil {
		return err
	}
	err = imaging.Save(img, outFilePath)
	return err
}
