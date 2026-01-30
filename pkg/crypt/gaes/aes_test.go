package gaes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// var aesKey = []byte("09876543poiuytre")
// var aesIV = []byte("12345678polkijhu")
var aseData = []byte("123987qwerttyiqweorqew")

// BenchmarkAES-8   	 1775088	       671.6 ns/op
func BenchmarkAES(b *testing.B) {
	decoder, err := NewEncrypter(aesKey, aesIV)
	if err != nil {
		b.Fatal(err)
	}
	encoder, err := NewDecrypter(aesKey, aesIV)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		ret := EnCrypt(aseData, encoder)
		/*d := */ DeCrypt(ret, decoder)
		//require.Equal(b, aseData, d)
	}
}

func TestEnCrypt(t *testing.T) {
	decoder, err := NewEncrypter(aesKey, aesIV)
	if err != nil {
		t.Fatal(err)
	}
	encoder, err := NewDecrypter(aesKey, aesIV)
	if err != nil {
		t.Fatal(err)
	}
	ret := EnCrypt(aseData, encoder)
	d := DeCrypt(ret, decoder)
	require.Equal(t, aseData, d)
}
