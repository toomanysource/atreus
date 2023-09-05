package ffmpegX

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/disintegration/imaging"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

var (
	ErrImageGenerate = errors.New("image generate error")
	ErrImageDecode   = errors.New("image decode error")
	ErrImageSave     = errors.New("image save error")
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
		return nil, errors.Join(ErrImageGenerate, err)
	}
	return buf, err
}

// SaveImage 保存jpeg格式的图片
// reader: 输入的图片流
// outFilePath: 输出文件路径
func SaveImage(reader io.Reader, outFilePath string) error {
	img, err := imaging.Decode(reader)
	if err != nil {
		return errors.Join(ErrImageDecode, err)
	}
	if err = imaging.Save(img, outFilePath); err != nil {
		return errors.Join(ErrImageSave, err)
	}
	return err
}
