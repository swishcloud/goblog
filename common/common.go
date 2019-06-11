package common

import (
	"crypto/md5"
	"encoding/hex"
)

func PwdCheck(hashedPwd string, rawPwd string) bool {
	return HashPwd(rawPwd)==hashedPwd
}

func HashPwd(rawPwd string) string {
	b := md5.Sum([]byte(rawPwd))
	hashedPassword := hex.EncodeToString(b[:])
	return hashedPassword
}
