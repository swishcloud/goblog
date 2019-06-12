package common

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/github-123456/gostudy/aesencryption"
)

func Lev2PwdCheck(hashedStr string, rawPwd string) bool {
	_, err := aesencryption.Decrypt(rawPwd, hashedStr)
	return err == nil
}
func PwdCheck(hashedPwd string, rawPwd string) bool {
	return HashPwd(rawPwd) == hashedPwd
}

func HashPwd(rawPwd string) string {
	b := md5.Sum([]byte(rawPwd))
	hashedPassword := hex.EncodeToString(b[:])
	return hashedPassword
}
func Md5Hash(str string) string {
	sb := []byte(str)
	b := md5.Sum(sb)
	return hex.EncodeToString(b[:])
}
