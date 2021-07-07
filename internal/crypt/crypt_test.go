package crypt

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestCrypt(t *testing.T) {
	viper.Reset()
	viper.Set("secrets.encryption", "mj%_V nq<{DIB:Lks7K+R]A6X?|-yJi/")
	data := []byte("testing crypt package")

	ciphertext, err := Encrypt(data)
	assert.NoError(t, err, "Failed encrypting data")

	plaintext, err := Decrypt(ciphertext)
	assert.NoError(t, err, "Failed decrypting data")

	assert.Equal(t, data, plaintext)
}

func TestCreateHash(t *testing.T) {
	viper.Reset()
	viper.Set("secrets.encryption", "test")

	hash := createHash()
	assert.Equal(t, 32, len(hash))
}

func BenchmarkEncrypt(b *testing.B) {
	viper.Reset()
	viper.Set("secrets.encryption", "uK=PxT1[7|3Nfv)eiW-pn5>M&CUh90'2")
	data := []byte("{id: OIsJZavECVXUhmTw, username: bench, birth_date: 2006-01-02T15:04:05Z, host: true}")

	for i := 0; i < b.N; i++ {
		Encrypt(data)
	}
}

func BenchmarkDecrypt(b *testing.B) {
	viper.Reset()
	viper.Set("secrets.encryption", "-X=A0WhdM>4BH)1w( }K*\\_:bU|qo3~Q")
	data := []byte("♣ŠyŠØÞÆõ‡0¦“UÚë‘Äà©,í?ðã‼ã.m³ò™Û¶m8Ò›”♣YÁË^ÆmŽc–rw®”l►q(H↔|%")

	for i := 0; i < b.N; i++ {
		Decrypt(data)
	}
}

func BenchmarkCreateHash(b *testing.B) {
	viper.Reset()
	viper.Set("secrets.encryption", "Jan!K+sbP3[q8{f\\h1I }TQ?C<aYj`|r")

	for i := 0; i < b.N; i++ {
		createHash()
	}
}
