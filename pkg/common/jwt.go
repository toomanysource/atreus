package common

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// ProduceToken 生成Token
func ProduceToken(tokenKey string, userId uint32, expired time.Duration) (string, error) {
	claims := jwt.MapClaims{}
	claims["user_id"] = userId
	expiredTime := time.Now().Add(expired)
	claims["exp"] = expiredTime.Unix()
	mySigningKey := []byte(tokenKey)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(mySigningKey)
}

// GenSaltPassword 生成一个含有盐值的密码字符串
func GenSaltPassword(salt, password string) string {
	// 创建一个 sha256 的哈希算法实例
	s1 := sha256.New()
	// 密码转化为字符数组
	s1.Write([]byte(password))
	// 使用 s1 进行哈希运算，并转化为字符串
	str1 := fmt.Sprintf("%x", s1.Sum(nil))

	// 创建另外一个 sha256 哈希算法，并且将 str1 和 salt 连接起来，转换为字符串，并且使用 s2 进行哈希运算
	s2 := sha256.New()
	s2.Write([]byte(str1 + salt))
	return fmt.Sprintf("%x", s2.Sum(nil))
}
