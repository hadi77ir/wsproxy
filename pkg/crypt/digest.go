package crypt

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"golang.org/x/crypto/sha3"
)

func GetDigestFunc(method string) func([]byte) []byte {
	switch method {
	case "sha1":
		return Sha1Sum
	case "sha256":
		return Sha256Sum
	case "sha224":
		return Sha224Sum
	case "sha384":
		return Sha384Sum
	case "sha512":
		return Sha512Sum
	case "sha3":
		return Sha3Sum
	}
	return nil
}

func Sha1Sum(in []byte) []byte {
	val := sha1.Sum(in)
	return val[:]
}

func Sha512Sum(in []byte) []byte {
	val := sha512.Sum512(in)
	return val[:]
}

func Sha384Sum(in []byte) []byte {
	val := sha512.Sum384(in)
	return val[:]
}

func Sha3Sum(in []byte) []byte {
	val := sha3.Sum512(in)
	return val[:]
}

func Sha256Sum(in []byte) []byte {
	val := sha256.Sum256(in)
	return val[:]
}

func Sha224Sum(in []byte) []byte {
	val := sha256.Sum224(in)
	return val[:]
}
