//go:build js && wasm

package result

import (
	"syscall/js"

	"github.com/samber/lo"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func (r *Result) MarshalJavaScript() js.Value {
	var sections []any
	sections = lo.Map(r.Sections, func(s *entity.Section, _ int) any {
		return s.MarshalJavaScript()
	})

	return js.ValueOf(map[string]any{
		"name":     r.Name,
		"size":     r.Size,
		"packages": r.Packages.MarshalJavaScript(),
		"sections": sections,
	})
}
