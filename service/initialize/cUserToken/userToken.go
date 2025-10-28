package cUserToken

import (
	"sun-panel/global"
	"sun-panel/lib/cache"

	"time"
)

func InitCUserToken() cache.Cacher[string] {
	// 设置为0表示永不过期
	return global.NewCache[string](0*time.Second, 48*time.Hour, "CUserToken")
}

// func InitVerifyCodeCachePool() {
// 	global.VerifyCodeCachePool = cache.NewGoCache(10*time.Minute, 60*time.Second)
// }
