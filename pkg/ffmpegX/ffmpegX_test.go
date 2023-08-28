package ffmpegX

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadFrameAsImage(t *testing.T) {
	_, err := ReadFrameAsImage("./test1.mp4", 1)
	assert.Nil(t, err)
}

func TestSaveImage(t *testing.T) {
	reader, _ := ReadFrameAsImage("./test1.mp4", 60)
	err := SaveImage(reader, "./test1.jpg")
	assert.Nil(t, err)
}
