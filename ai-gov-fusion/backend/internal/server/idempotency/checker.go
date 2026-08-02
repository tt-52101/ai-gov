package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Checker 是 fund.IdempotencyChecker 接口的内存实现。
// 它提供幂等键的 Claim/Store/Retrieve/Release 操作，使用内存 map 存储状态。
//
// 注意：此实现是进程内临时的——重启后所有状态丢失。
// 资金操作的持久幂等记录由 fund.Store.CheckIdempotency / StoreIdempotency
// 方法在数据库事务内处理。
type Checker struct {
	mu      sync.RWMutex
	keys    map[string]bool   // key -> claimed
	results map[string]string // key -> JSON result
}

// NewChecker 创建一个新的内存幂等检查器。
func NewChecker() *Checker {
	return &Checker{
		keys:    make(map[string]bool),
		results: make(map[string]string),
	}
}

// Claim 原子地申请一个幂等键。若键尚未被 Claim 则返回 true，
// 若已存在则返回 false。
func (c *Checker) Claim(ctx context.Context, key string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.keys[key] {
		return false, nil
	}
	c.keys[key] = true
	return true, nil
}

// Store 将结果与幂等键关联存储。结果必须是可 JSON 序列化的。
func (c *Checker) Store(ctx context.Context, key string, result any) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("idempotency: 序列化结果失败: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results[key] = string(data)
	return nil
}

// Retrieve 获取先前存储的结果。若结果存在则解码到 result 参数并返回 true。
func (c *Checker) Retrieve(ctx context.Context, key string, result any) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, ok := c.results[key]
	if !ok {
		return false, nil
	}
	if err := json.Unmarshal([]byte(data), result); err != nil {
		return false, fmt.Errorf("idempotency: 反序列化结果失败: %w", err)
	}
	return true, nil
}

// Release 释放已 Claim 但尚未 Store 的幂等键。
// 若键已有存储结果则不执行任何操作（Store 已被调用）。
func (c *Checker) Release(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 仅在没有存储结果时才删除。
	if _, hasResult := c.results[key]; !hasResult {
		delete(c.keys, key)
	}
	return nil
}
