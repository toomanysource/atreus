package common

import (
	"crypto/md5"
	"math/big"

	"github.com/google/uuid"
)

func NewUUIDInt() uint64 {
	// 解析UUID
	uuidByte := uuid.New().NodeID()

	hasher := md5.New()
	hasher.Write(uuidByte)
	hashByte := hasher.Sum(nil)

	// 将哈希值转换为大整数
	hashBigInt := big.NewInt(0)
	hashBigInt.SetBytes(hashByte[:])

	// 将大整数转换为int64类型
	return hashBigInt.Uint64()
}
