package libkademlia

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	mathrand "math/rand"
	"time"
	"sss"
)

type VanashingDataObject struct {
	AccessKey  int64
	Ciphertext []byte
	NumberKeys byte
	Threshold  byte
}

func GenerateRandomCryptoKey() (ret []byte) {
	for i := 0; i < 32; i++ {
		ret = append(ret, uint8(mathrand.Intn(256)))
	}
	return
}

func GenerateRandomAccessKey() (accessKey int64) {
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	accessKey = r.Int63()
	return
}

func CalculateSharedKeyLocations(accessKey int64, count int64) (ids []ID) {
	r := mathrand.New(mathrand.NewSource(accessKey))
	ids = make([]ID, count)
	for i := int64(0); i < count; i++ {
		for j := 0; j < IDBytes; j++ {
			ids[i][j] = uint8(r.Intn(256))
		}
	}
	return
}

func encrypt(key []byte, text []byte) (ciphertext []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	ciphertext = make([]byte, aes.BlockSize+len(text))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], text)
	return
}

func decrypt(key []byte, ciphertext []byte) (text []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext is not long enough")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext
}

func (k *Kademlia) VanishData(data []byte, numberKeys byte,
	threshold byte, timeoutSeconds int) (vdo VanashingDataObject) {
	cryptographic_key := GenerateRandomCryptoKey()
	data_after_encrypt := encrypt(cryptographic_key, data)
	sss_key_map, _ := sss.Split(numberKeys, threshold, cryptographic_key)
	// xiangyu: handle error here??
	access_key := GenerateRandomAccessKey()
	count := int64(numberKeys)
	node_list_to_store := CalculateSharedKeyLocations(access_key, count)
	node_index := 0
	for key, value := range sss_key_map {
		all := append([]byte{key}, value...)
		// xiangyu : start new go_routine to store?
		k.DoIterativeStore(node_list_to_store[node_index], all)
		node_index += 1
	}
	VDO_obj := VanashingDataObject{
		AccessKey: access_key,
		Ciphertext: data_after_encrypt,
		NumberKeys: numberKeys,
		Threshold: threshold,
	}
	return VDO_obj
}

func (k *Kademlia) UnvanishData(vdo VanashingDataObject) (data []byte) {
	return nil
}
