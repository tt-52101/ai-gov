package routing

import (
	"fmt"
	"sort"
	"sync"
)

// registryMutex 保护全局策略注册表免受并发访问。
var registryMutex sync.RWMutex

// registry 全局策略注册表，以策略 ID 为键。
var registry = make(map[string]Strategy)

// Register 向全局注册表注册一个策略。
// 如果策略 ID 已存在，返回错误。该函数线程安全。
func Register(s Strategy) error {
	if s == nil {
		return fmt.Errorf("routing: 不能注册 nil 策略")
	}
	if s.ID() == "" {
		return fmt.Errorf("routing: 策略 ID 不能为空")
	}

	registryMutex.Lock()
	defer registryMutex.Unlock()

	if _, exists := registry[s.ID()]; exists {
		return fmt.Errorf("routing: 策略 %q 已注册", s.ID())
	}
	registry[s.ID()] = s
	return nil
}

// GetStrategy 按 ID 查找已注册的策略。
// 未找到返回 nil。调用方应检查 nil。
func GetStrategy(id string) Strategy {
	registryMutex.RLock()
	defer registryMutex.RUnlock()
	return registry[id]
}

// GetRegistered 返回所有已注册策略的 ID 列表，按字母序排序。
func GetRegistered() []string {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// HasStrategy 检查指定 ID 的策略是否已注册。
func HasStrategy(id string) bool {
	registryMutex.RLock()
	defer registryMutex.RUnlock()
	_, ok := registry[id]
	return ok
}
