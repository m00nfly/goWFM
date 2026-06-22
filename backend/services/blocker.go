package services

import (
	"net"
	"sync"
	"time"

	"goWFM/config"
)

type failureRecord struct {
	count     int
	firstFail time.Time
	blockedAt time.Time
	blocked   bool
}

// Blocker 管理 IP 和账号封锁状态
type Blocker struct {
	mu       sync.Mutex
	ipFails  map[string]*failureRecord
	accFails map[string]*failureRecord
}

// GlobalBlocker 全局封锁引擎实例
var GlobalBlocker = &Blocker{
	ipFails:  make(map[string]*failureRecord),
	accFails: make(map[string]*failureRecord),
}

// IsIPBlocked 检查 IP 是否被封锁
func (b *Blocker) IsIPBlocked(ip string) bool {
	sec := config.GetSecurity()
	if !sec.IPBlockEnabled {
		return false
	}

	// 白名单免检
	if b.IsWhitelisted(ip, sec.WhitelistIPs) {
		return false
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	rec, exists := b.ipFails[ip]
	if !exists {
		return false
	}
	if !rec.blocked {
		return false
	}

	// 检查封锁是否过期
	if time.Since(rec.blockedAt) > time.Duration(sec.IPBlockDuration)*time.Second {
		delete(b.ipFails, ip)
		return false
	}
	return true
}

// IsAccountBlocked 检查账号是否被封锁
func (b *Blocker) IsAccountBlocked(username string) bool {
	sec := config.GetSecurity()
	if !sec.AccountBlockEnabled {
		return false
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	rec, exists := b.accFails[username]
	if !exists {
		return false
	}
	if !rec.blocked {
		return false
	}

	// 检查封锁是否过期
	if time.Since(rec.blockedAt) > time.Duration(sec.AccountBlockDuration)*time.Second {
		delete(b.accFails, username)
		return false
	}
	return true
}

// RecordFailure 记录一次登录失败
func (b *Blocker) RecordFailure(ip, username string) {
	sec := config.GetSecurity()

	b.mu.Lock()
	defer b.mu.Unlock()

	// IP 封锁逻辑
	if sec.IPBlockEnabled && !b.isWhitelistedLocked(ip, sec.WhitelistIPs) {
		rec, exists := b.ipFails[ip]
		if !exists {
			rec = &failureRecord{firstFail: time.Now()}
			b.ipFails[ip] = rec
		}

		// 检查时间窗口
		if time.Since(rec.firstFail) > time.Duration(sec.IPBlockWindow)*time.Second {
			rec.count = 0
			rec.firstFail = time.Now()
		}

		rec.count++
		if rec.count >= sec.IPBlockMaxFailures {
			rec.blocked = true
			rec.blockedAt = time.Now()
		}
	}

	// 账号封锁逻辑
	if sec.AccountBlockEnabled && username != "" {
		rec, exists := b.accFails[username]
		if !exists {
			rec = &failureRecord{firstFail: time.Now()}
			b.accFails[username] = rec
		}

		if time.Since(rec.firstFail) > time.Duration(sec.AccountBlockWindow)*time.Second {
			rec.count = 0
			rec.firstFail = time.Now()
		}

		rec.count++
		if rec.count >= sec.AccountBlockMaxFails {
			rec.blocked = true
			rec.blockedAt = time.Now()
		}
	}
}

// ResetOnSuccess 登录成功时重置计数
func (b *Blocker) ResetOnSuccess(ip, username string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.ipFails, ip)
	if username != "" {
		delete(b.accFails, username)
	}
}

// IsWhitelisted 检查 IP 是否在白名单中（支持 IP 和 CIDR）
func (b *Blocker) IsWhitelisted(ip string, whitelist []string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, entry := range whitelist {
		// 尝试解析为 CIDR
		if _, subnet, err := net.ParseCIDR(entry); err == nil {
			if subnet.Contains(parsedIP) {
				return true
			}
			continue
		}
		// 直接 IP 匹配
		if entry == ip {
			return true
		}
	}
	return false
}

// isWhitelistedLocked 内部使用（已持有锁时调用）
func (b *Blocker) isWhitelistedLocked(ip string, whitelist []string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	for _, entry := range whitelist {
		if _, subnet, err := net.ParseCIDR(entry); err == nil {
			if subnet.Contains(parsedIP) {
				return true
			}
			continue
		}
		if entry == ip {
			return true
		}
	}
	return false
}

// Cleanup 清理过期的封锁记录
func (b *Blocker) Cleanup() {
	sec := config.GetSecurity()

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()

	for ip, rec := range b.ipFails {
		if rec.blocked {
			if now.Sub(rec.blockedAt) > time.Duration(sec.IPBlockDuration)*time.Second {
				delete(b.ipFails, ip)
			}
		} else {
			if now.Sub(rec.firstFail) > time.Duration(sec.IPBlockWindow)*time.Second {
				delete(b.ipFails, ip)
			}
		}
	}

	for acc, rec := range b.accFails {
		if rec.blocked {
			if now.Sub(rec.blockedAt) > time.Duration(sec.AccountBlockDuration)*time.Second {
				delete(b.accFails, acc)
			}
		} else {
			if now.Sub(rec.firstFail) > time.Duration(sec.AccountBlockWindow)*time.Second {
				delete(b.accFails, acc)
			}
		}
	}
}

// StartBlockerCleanup 启动后台定期清理协程
func StartBlockerCleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		GlobalBlocker.Cleanup()
	}
}
