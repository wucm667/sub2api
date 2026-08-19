package service

import (
	"bytes"
	"sort"
	"strings"

	"github.com/tidwall/gjson"
)

const (
	// 工具定义在多轮历史里最多再嵌套一层 tools，留出余量后截断，避免畸形请求体
	// 造成无界递归。
	openAIResponsesToolSchemaMaxDepth = 4
	// JSON Schema 里 type 只能是字符串或字符串数组；显式 null 无论哪个方言都非法，
	// 补成 object 与 upstream 对该工具的实际期望一致。
	openAIResponsesToolSchemaFallbackType = `"object"`
	// 显式 null 在 JSON 里只有这一种字面量形态。
	openAIResponsesToolSchemaNullLiteral = "null"
)

// openAIResponsesToolSchemaNullType 记录一处待修正的 Schema 片段，用原始 body
// 上的绝对字节偏移表示，便于最后一次性拼接。
type openAIResponsesToolSchemaNullType struct {
	offset      int
	length      int
	replacement string
}

// sanitizeOpenAIResponsesToolParameterTypes 修正请求体中 OpenAI 不接受的工具根 Schema。
//
// Codex Desktop 内置的 automation_update 工具会带 parameters.type = null，
// OpenAI 直接回 400 invalid_function_parameters，而网关把该状态归一成可重试的
// 502 upstream_error；该工具定义又会沉进多轮历史，导致之后每一轮继续失败并在
// 账号池里反复重放同一份坏 Schema。
//
// 普通工具仍只修正显式 null。已知的 automation_update 名称若缺失根 type 且使用
// oneOf/anyOf，则在所有分支均为 object 时保留 union 并补根 type；否则退回宽松 object
// 并仅关闭该工具的 strict。其他缺失 type 的 Schema 保持原样。
//
// 实现上先收集全部命中的绝对偏移，再一次性拼出新 body：逐个 sjson.SetBytes 每次
// 都会重扫并全量拷贝整个文档，命中 N 处就是 N 次全量拷贝，而 /v1/responses 的
// body 上限是 gateway.max_body_size（默认 256MB），构造请求能塞进百万级命中。
func sanitizeOpenAIResponsesToolParameterTypes(body []byte) ([]byte, bool, error) {
	return sanitizeOpenAIResponsesToolParameterTypesAtDepth(body, 0, "")
}

func sanitizeOpenAIResponsesToolParameterTypesAtDepth(body []byte, depth int, namespace string) ([]byte, bool, error) {
	if len(body) == 0 {
		return body, false, nil
	}

	hits := make([]openAIResponsesToolSchemaNullType, 0, 2)
	collectOpenAIResponsesToolSchemaNullTypes(body, gjson.GetBytes(body, "tools"), depth, namespace, &hits)
	if input := gjson.GetBytes(body, "input"); input.IsArray() {
		input.ForEach(func(_, item gjson.Result) bool {
			if item.IsObject() {
				collectOpenAIResponsesToolSchemaNullTypes(body, item.Get("tools"), depth, namespace, &hits)
			}
			return true
		})
	}
	if len(hits) == 0 {
		return body, false, nil
	}

	// tools 与 input 在 body 里的先后顺序由客户端决定，收集顺序不保证单调。
	sort.Slice(hits, func(i, j int) bool { return hits[i].offset < hits[j].offset })

	sanitized := make([]byte, 0, len(body)+len(hits)*len(openAIResponsesToolSchemaFallbackType))
	cursor := 0
	for _, hit := range hits {
		// 收集阶段已逐个校验过区间，这里再挡一次重叠，保证拼接严格单调向前。
		if hit.offset < cursor {
			continue
		}
		sanitized = append(sanitized, body[cursor:hit.offset]...)
		sanitized = append(sanitized, hit.replacement...)
		cursor = hit.offset + hit.length
	}
	sanitized = append(sanitized, body[cursor:]...)
	return sanitized, true, nil
}

// collectOpenAIResponsesToolSchemaNullTypes 收集一个 tools 数组里所有需要修正的
// Schema 位置。显式 null 不按 tool type 过滤；缺失根 type 的 union 仅识别已知的
// automation_update 名称。
func collectOpenAIResponsesToolSchemaNullTypes(
	body []byte, tools gjson.Result, depth int, namespace string, hits *[]openAIResponsesToolSchemaNullType,
) {
	if depth > openAIResponsesToolSchemaMaxDepth || !tools.IsArray() {
		return
	}
	tools.ForEach(func(_, tool gjson.Result) bool {
		if !tool.IsObject() {
			return true
		}
		toolNamespace := namespace
		if tool.Get("type").String() == "namespace" {
			toolNamespace = tool.Get("name").String()
		}
		// Responses 形态用顶层 parameters，ChatCompletions 形态用 function.parameters，
		// 两种都可能出现在 Responses 请求里（见 normalizeCodexTools）。
		for _, suffix := range []string{"parameters", "function.parameters"} {
			params := tool.Get(suffix)
			if !params.IsObject() {
				continue
			}
			// gjson 用 Type==Null 同时表示「显式 null」和「路径不存在」，靠 Raw
			// 区分：不存在时 Raw 为空串。
			if typ := params.Get("type"); typ.Type == gjson.Null && typ.Raw == openAIResponsesToolSchemaNullLiteral {
				appendOpenAIResponsesToolSchemaEdit(body, typ, openAIResponsesToolSchemaFallbackType, hits)
				continue
			}
			name := tool.Get("name").String()
			if suffix == "function.parameters" {
				name = tool.Get("function.name").String()
			}
			if !params.Get("type").Exists() && isAutomationUpdateTool(name, namespace) {
				hasUnion, allObject := openAIResponsesUnionsHaveObjectRoot(params)
				if hasUnion {
					if allObject {
						appendOpenAIResponsesToolSchemaEdit(body, params, insertObjectType(params.Raw), hits)
					} else {
						appendOpenAIResponsesToolSchemaEdit(body, tool, fallbackAutomationUpdateTool(tool, params, suffix, depth, namespace), hits)
					}
				}
			}
		}
		// 历史输入里的工具定义会再嵌套一层 tools（upstream 报错路径形如
		// input[234].tools[0].tools[3].parameters）。
		collectOpenAIResponsesToolSchemaNullTypes(body, tool.Get("tools"), depth+1, toolNamespace, hits)
		return true
	})
}

// appendOpenAIResponsesToolSchemaEdit 先校验 gjson 给出的偏移确实指向原始
// body 上目标片段，再记录。gjson 对嵌套取值同样返回相对原始文档的绝对偏移，但
// Index 为 0 表示未知；偏移不可用时跳过该处而不是猜位置——少修一个工具只是维持
// 现状，拼错位置会损坏整个请求体。
func appendOpenAIResponsesToolSchemaEdit(
	body []byte, value gjson.Result, replacement string, hits *[]openAIResponsesToolSchemaNullType,
) {
	end := value.Index + len(value.Raw)
	if value.Index <= 0 || end > len(body) {
		return
	}
	if !bytes.Equal(body[value.Index:end], []byte(value.Raw)) {
		return
	}
	*hits = append(*hits, openAIResponsesToolSchemaNullType{offset: value.Index, length: len(value.Raw), replacement: replacement})
}

func isAutomationUpdateTool(name, namespace string) bool {
	return name == "automation_update" || name == "codex_app__automation_update" ||
		name == "codex_app.automation_update" || (namespace == "codex_app" && name == "automation_update")
}

func openAIResponsesUnionsHaveObjectRoot(params gjson.Result) (bool, bool) {
	hasUnion := false
	for _, keyword := range []string{"oneOf", "anyOf"} {
		union := params.Get(keyword)
		if !union.Exists() {
			continue
		}
		hasUnion = true
		if !union.IsArray() {
			return true, false
		}
		valid := true
		count := 0
		union.ForEach(func(_, branch gjson.Result) bool {
			count++
			typ := branch.Get("type")
			if !branch.IsObject() || typ.Type != gjson.String || typ.String() != "object" {
				valid = false
			}
			return valid
		})
		if !valid || count == 0 {
			return true, false
		}
	}
	return hasUnion, hasUnion
}

func insertObjectType(raw string) string {
	return `{"type":"object",` + strings.TrimPrefix(raw, "{")
}

func insertStrictFalse(raw string) string {
	return `{"strict":false,` + strings.TrimPrefix(raw, "{")
}

func fallbackAutomationUpdateTool(tool, params gjson.Result, paramsPath string, depth int, namespace string) string {
	const fallback = `{"type":"object","additionalProperties":true}`
	start := params.Index - tool.Index
	raw := tool.Raw[:start] + fallback + tool.Raw[start+len(params.Raw):]
	strictPath := "strict"
	if paramsPath == "function.parameters" {
		strictPath = "function.strict"
	}
	raw = setOpenAIResponsesToolStrictFalse(raw, strictPath)

	// This replacement covers the entire tool, so descendant edits collected from
	// the original body would overlap it and be skipped by the final splice. Repair
	// the replacement itself to preserve nested tools and function.parameters fixes.
	wrapper := []byte(`{"tools":[` + raw + `]}`)
	sanitized, changed, _ := sanitizeOpenAIResponsesToolParameterTypesAtDepth(wrapper, depth, namespace)
	if !changed {
		return raw
	}
	return gjson.GetBytes(sanitized, "tools.0").Raw
}

func setOpenAIResponsesToolStrictFalse(raw, strictPath string) string {
	strict := gjson.Get(raw, strictPath)
	if strict.Exists() {
		return raw[:strict.Index] + "false" + raw[strict.Index+len(strict.Raw):]
	}
	if strictPath == "strict" {
		return insertStrictFalse(raw)
	}

	function := gjson.Get(raw, "function")
	if !function.IsObject() {
		return raw
	}
	replacement := insertStrictFalse(function.Raw)
	return raw[:function.Index] + replacement + raw[function.Index+len(function.Raw):]
}
