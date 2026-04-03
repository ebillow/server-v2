package gaes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

func NewEncrypter(key []byte, iv []byte) (cipher.BlockMode, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return cipher.NewCBCEncrypter(block, iv), nil
}

func NewDecrypter(key []byte, iv []byte) (cipher.BlockMode, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return cipher.NewCBCDecrypter(block, iv), nil
}

// 加密函数
func EnCrypt(src []byte, blockMode cipher.BlockMode) []byte {
	// 明文组数据填充
	paddingText := PKCS7Padding(src, blockMode.BlockSize())
	// 加密
	// dst := make([]byte, len(paddingText))
	blockMode.CryptBlocks(paddingText, paddingText)
	return paddingText
}

func DeCrypt(src []byte, blockMode cipher.BlockMode) ([]byte, error) {
	// 解密
	// dst := make([]byte, len(src))
	blockMode.CryptBlocks(src, src)
	// 分组移除
	return PKCS7UnPadding(src)
}

// PKCS7Padding 填充 (这里 append 会分配新内存，不会污染源数据，是安全的)
func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// PKCS7UnPadding 移除填充 (增加严格的安全校验)
func PKCS7UnPadding(plaintext []byte) ([]byte, error) {
	length := len(plaintext)
	if length == 0 {
		return nil, errors.New("plaintext is empty")
	}

	unpadding := int(plaintext[length-1])

	// 校验 1：填充长度不能大于数据总长度，也不能为 0
	if unpadding > length || unpadding == 0 {
		return nil, errors.New("invalid padding length")
	}

	// 校验 2：严格校验所有填充字节是否一致 (防止恶意篡改导致的逻辑漏洞)
	padStart := length - unpadding
	for i := padStart; i < length; i++ {
		if plaintext[i] != byte(unpadding) {
			return nil, errors.New("invalid padding bytes")
		}
	}

	return plaintext[:padStart], nil
}
