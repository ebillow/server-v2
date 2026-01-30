/***********************************************************
 *
 * Diffie–Hellman key exchange
 *
 * 1. Alice and Bob agree to use a prime number p = 23 and base g = 5.
 *
 * 2. Alice chooses a secret integer a = 6, then sends Bob A = g^a mod p
 * 		A = 5^6 mod 23
 * 		A = 15,625 mod 23
 * 		A = 8
 *
 * 3. Bob chooses a secret integer b = 15, then sends Alice B = g^b mod p
 * 		B = 5^15 mod 23
 * 		B = 30,517,578,125 mod 23
 * 		B = 19
 *
 * 4. Alice computes s = B^a mod p
 * 		s = 19^6 mod 23
 * 		s = 47,045,881 mod 23
 * 		s = 2
 *
 * 5. Bob computes s = A^b mod p
 *	 	s = 8^15 mod 23
 * 		s = 35,184,372,088,832 mod 23
 * 		s = 2
 *
 * 6. Alice and Bob now share a secret (the number 2) because 6 × 15 is the same as 15 × 6
 */
package dh

import (
	"math"
	"math/big"
	"math/rand"
	"time"
)

var (
	min = rand.New(rand.NewSource(time.Now().UnixNano()))
	max = big.NewInt(math.MaxInt64)
	g   = big.NewInt(2)
	p   = big.NewInt(987123654)
)

// Exchange 计算privateKey , publicKey
func Exchange() (*big.Int, *big.Int) {
	privateKey := big.NewInt(0).Rand(min, max)
	publicKey := big.NewInt(0).Exp(g, privateKey, p)
	return privateKey, publicKey
}

// GetKey 得到key
func GetKey(privateKey, publicKey *big.Int) *big.Int {
	return big.NewInt(0).Exp(publicKey, privateKey, p)
}
