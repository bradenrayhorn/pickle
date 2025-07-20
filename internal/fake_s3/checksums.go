package fakes3

import (
	"crypto/sha256"
	"hash/crc32"
)

func GetChecksums(data []byte) ([]byte, []byte) {
	sha256 := sha256.Sum256(data)
	crc32c := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	_, err := crc32c.Write(data)
	if err != nil {
		panic(err)
	}

	return crc32c.Sum(nil), sha256[:]
}
