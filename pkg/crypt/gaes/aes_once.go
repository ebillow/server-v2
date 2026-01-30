package gaes

import (
	"crypto/aes"
	"errors"
	"go.uber.org/zap"
)

var aesKey = []byte("56pol1234kij78hu")
var aesIV = []byte("093po54iuy876tre")

func Init(key string, iv string) bool {
	if key == "" || iv == "" {
		zap.L().Info("使用默认aes key")
		return true
	}

	aesKey = []byte(key)
	aesIV = []byte(iv)

	if len(aesKey) != 16 || len(iv) != 16 {
		zap.L().Error("ase key or iv len err")
		return false
	}
	return true
}

// EnCryptOnce 加密一次
func EnCryptOnce(src []byte) ([]byte, error) {
	blockMode, err := NewEncrypter(aesKey, aesIV)
	if err != nil {
		return nil, err
	}
	// 明文组数据填充
	paddingText := PKCS7Padding(src, blockMode.BlockSize())
	// 加密
	// fmt.Println("pkcs7:", paddingText)
	// dst := make([]byte, len(paddingText))
	blockMode.CryptBlocks(paddingText, paddingText)
	return paddingText, nil
}

// DeCryptOnce 解密一次
func DeCryptOnce(src []byte) ([]byte, error) {
	//  基础长度检查
	l := len(src)
	if l < aes.BlockSize || l%aes.BlockSize != 0 {
		return nil, errors.New("invalid ciphertext length")
	}

	blockMode, err := NewDecrypter(aesKey, aesIV)
	if err != nil {
		return nil, err
	}
	// 解密
	// dst := make([]byte, len(src))
	blockMode.CryptBlocks(src, src)
	// 分组移除
	return PKCS7UnPadding(src), nil
}
