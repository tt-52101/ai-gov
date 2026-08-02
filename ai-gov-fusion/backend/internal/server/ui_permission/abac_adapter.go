package ui_permission

import (
	"context"

	"tokenhub/backend/internal/server/abac"
)

// abacEngineAdapter 将 *abac.Engine 适配为 ui_permission.ABACEngine 接口。
// abac.Engine.Evaluate 接受 abac.Subject / abac.Resource，
// 而 ui_permission.ABACEngine 要求 ui_permission.Subject / ui_permission.Resource。
// 此适配器执行结构体字段的转换。
type abacEngineAdapter struct {
	engine *abac.Engine
}

// NewABACAdapter 创建一个将 abac.Engine 包装为 ui_permission.ABACEngine 的适配器。
func NewABACAdapter(engine *abac.Engine) ABACEngine {
	return &abacEngineAdapter{engine: engine}
}

// Evaluate 将 ui_permission 类型转换为 abac 类型，委托给底层引擎，然后返回结果。
func (a *abacEngineAdapter) Evaluate(ctx context.Context, subject Subject, action string, resource Resource) error {
	abacSubject := abac.Subject{
		Type: subject.Type,
		ID:   subject.ID,
	}
	abacResource := abac.Resource{
		Type: resource.Type,
		ID:   resource.ID,
	}
	return a.engine.Evaluate(ctx, abacSubject, action, abacResource)
}
