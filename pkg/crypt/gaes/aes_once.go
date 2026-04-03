package gaes

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"go.uber.org/zap"
)

var (
	aesKey = []byte("56pol1234kij78hu")
	aesIV  = []byte("093po54iuy876tre")
	// 全局缓存 cipher.Block，极大地提升性能
	globalBlock cipher.Block
)

func Init(key string, iv string) error {
	if key != "" {
		aesKey = []byte(key)
	}
	if iv != "" {
		aesIV = []byte(iv)
	}

	// AES 支持 16, 24, 32 字节的 key
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		zap.L().Error("aes.NewCipher failed", zap.Error(err))
		return err
	}

	if len(aesIV) != block.BlockSize() {
		err := errors.New("IV length must equal block size")
		zap.L().Error("iv len error", zap.Error(err))
		return err
	}

	globalBlock = block
	return nil
}

// 获取全局 Block，确保 fallback 机制
func getBlock() cipher.Block {
	if globalBlock == nil {
		// 兜底：如果没有调用 Init，使用默认 key 初始化
		globalBlock, _ = aes.NewCipher(aesKey)
	}
	return globalBlock
}

// EnCryptOnce 加密一次
func EnCryptOnce(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}

	block := getBlock()
	blockSize := block.BlockSize()

	// 填充明文
	paddingText := PKCS7Padding(src, blockSize)

	// 分配新的切片，避免污染入参 src (生产环境安全第一)
	dst := make([]byte, len(paddingText))

	// 每次加密必须新建 CBCEncrypter，因为它是带有内部状态的
	blockMode := cipher.NewCBCEncrypter(block, aesIV)
	blockMode.CryptBlocks(dst, paddingText)

	return dst, nil
}

// DeCryptOnce 解密一次
func DeCryptOnce(src []byte) ([]byte, error) {
	block := getBlock()
	blockSize := block.BlockSize()

	// 基础长度检查
	l := len(src)
	if l == 0 || l%blockSize != 0 {
		return nil, errors.New("invalid ciphertext length")
	}

	// 分配新的切片，避免污染入参
	dst := make([]byte, l)

	blockMode := cipher.NewCBCDecrypter(block, aesIV)
	blockMode.CryptBlocks(dst, src)

	// 分组移除，并处理错误
	return PKCS7UnPadding(dst)
}
