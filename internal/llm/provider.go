package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Config struct {
	Provider string
	Model    string
	BaseURL  string
}

type GradeItem struct {
	GUID  string
	Title string
	Meta  string
}

type GradeRequest struct {
	SourceContext      string
	Context            string
	GlobalHigh         string
	GlobalInterest     string
	GlobalUninterested string
	GlobalAvoid        string
	High               string
	Interest           string
	Uninterested       string
	Avoid              string
	Items              []GradeItem
	WriteGradeResults  func(context.Context, []GradeResult) error
}

type GradeResult struct {
	GUID   string
	Level  string
	Reason string
}

type SummaryRequest struct {
	PreferredLanguage string
	SourceContext     string
	Context           string
	GUID              string
	Title             string
	Description       string
	Extra             map[string]any
	WriteSummary      func(context.Context, SummaryResult) error
}

type SummaryResult struct {
	GUID        string
	Title       string
	Description string
	Rejected    bool
}

type Provider interface {
	Grade(ctx context.Context, req GradeRequest) ([]GradeResult, error)
	Summarize(ctx context.Context, req SummaryRequest) (SummaryResult, error)
}

type QwenProvider struct {
	Config Config
}

func CreateProvider(cfg Config) (Provider, error) {
	switch cfg.Provider {
	case "qwen":
		if os.Getenv(providerEnvKeys[cfg.Provider]) == "" {
			return nil, fmt.Errorf("%s not set", providerEnvKeys[cfg.Provider])
		}
		return QwenProvider{Config: cfg}, nil
	default:
		return nil, fmt.Errorf("unsupported provider %q", cfg.Provider)
	}
}

var providerEnvKeys = map[string]string{
	"anthropic":  "ANTHROPIC_API_KEY",
	"openai":     "OPENAI_API_KEY",
	"gemini":     "GEMINI_API_KEY",
	"qwen":       "QWEN_API_KEY",
	"openrouter": "OPENROUTER_API_KEY",
	"grok":       "GROK_API_KEY",
}

func (p QwenProvider) Grade(ctx context.Context, req GradeRequest) ([]GradeResult, error) {
	type responseEnvelope struct {
		Items []GradeResult `json:"items"`
	}
	var envelope responseEnvelope
	if err := p.callTool(ctx, chatCompletionRequest{
		prompt:   buildGradePrompt(req),
		toolName: "write_grade_results",
		toolSpec: gradeResultsToolDefinition(),
	}, &envelope); err != nil {
		return nil, err
	}
	if req.WriteGradeResults != nil {
		if err := req.WriteGradeResults(ctx, envelope.Items); err != nil {
			return nil, err
		}
	}
	return envelope.Items, nil
}

func (p QwenProvider) Summarize(ctx context.Context, req SummaryRequest) (SummaryResult, error) {
	var result SummaryResult
	if err := p.callTool(ctx, chatCompletionRequest{
		prompt:   buildSummaryPrompt(req),
		toolName: "write_summary",
		toolSpec: summaryToolDefinition(),
	}, &result); err != nil {
		return SummaryResult{}, err
	}
	if req.GUID != "" {
		result.GUID = req.GUID
	}
	if req.WriteSummary != nil {
		if err := req.WriteSummary(ctx, result); err != nil {
			return SummaryResult{}, err
		}
	}
	return result, nil
}

type chatCompletionRequest struct {
	prompt   string
	toolName string
	toolSpec map[string]any
}

func (p QwenProvider) callTool(ctx context.Context, input chatCompletionRequest, target any) error {
	baseURL := strings.TrimRight(p.Config.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}

	payload := map[string]any{
		"model": p.Config.Model,
		"messages": []map[string]any{
			{"role": "user", "content": input.prompt},
		},
		"tools": []map[string]any{
			input.toolSpec,
		},
		"tool_choice": map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": input.toolName,
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv(providerEnvKeys["qwen"]))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qwen request failed: %s", strings.TrimSpace(string(data)))
	}

	var completion struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return err
	}
	if len(completion.Choices) == 0 {
		return fmt.Errorf("qwen response contained no choices")
	}
	toolCalls := completion.Choices[0].Message.ToolCalls
	if len(toolCalls) != 1 {
		return fmt.Errorf("expected exactly one tool call, got %d", len(toolCalls))
	}
	call := toolCalls[0]
	if call.Function.Name != input.toolName {
		return fmt.Errorf("expected tool call %q, got %q", input.toolName, call.Function.Name)
	}
	if call.Function.Arguments == "" {
		return fmt.Errorf("tool call %q returned empty arguments", input.toolName)
	}
	return json.Unmarshal([]byte(call.Function.Arguments), target)
}

func gradeResultsToolDefinition() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "write_grade_results",
			"description": "Write the final grade results for all items as structured output.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"items": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"guid": map[string]any{
									"type": "string",
								},
								"level": map[string]any{
									"type": "string",
									"enum": []string{"critical", "recommended", "optional", "rejected"},
								},
								"reason": map[string]any{
									"type": "string",
								},
							},
							"required":             []string{"guid", "level", "reason"},
							"additionalProperties": false,
						},
					},
				},
				"required":             []string{"items"},
				"additionalProperties": false,
			},
		},
	}
}

func summaryToolDefinition() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "write_summary",
			"description": "Write the final summary result for the current item as structured output.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"guid": map[string]any{
						"type": "string",
					},
					"title": map[string]any{
						"type": "string",
					},
					"description": map[string]any{
						"type": "string",
					},
					"rejected": map[string]any{
						"type": "boolean",
					},
				},
				"required":             []string{"guid", "title", "description", "rejected"},
				"additionalProperties": false,
			},
		},
	}
}

func buildGradePrompt(req GradeRequest) string {
	lines := make([]string, 0, len(req.Items)+12)
	lines = append(lines,
		"# 任务：内容分级",
		"",
		"你必须调用 `write_grade_results` 工具完成输出。",
		"禁止输出普通文本、禁止输出 JSON、禁止解释过程。",
		"只允许调用一次工具。",
		`工具参数必须是：{"items":[{"guid":"string","level":"critical|recommended|optional|rejected","reason":"string"}]}`,
		"",
		"根据用户兴趣配置，基于标题和摘要对文章进行分级。",
		"",
		"## 分级规则",
		"",
		"按以下步骤进行分级：",
		"",
		"1. **理解用户兴趣**：",
		"   - 兴趣分为四类：high_interest（很感兴趣）、interest（感兴趣）、uninterested（不太感兴趣）、avoid（想要避开）",
		"   - 全局配置和来源配置都要考虑，来源配置权重更高",
		"",
		"2. **进行分级**：根据文章的标题、摘要，判断文章主题，再判断用户对该主题的兴趣程度",
		"   - **level 字段的 4 个合法值（必须精确匹配）**：",
		"     - critical：用户会强烈感兴趣，必看内容",
		"     - recommended：用户会感兴趣，推荐阅读",
		"     - optional：标题含义模糊或兴趣不明确，可选",
		"     - rejected：用户不感兴趣，应该被排除",
		"   - **综合判断示例**：",
		"     - 例 1：文章主要讲主题 A，顺便提到主题 B → 主要主题是 A → 即使全局配置 B 是 high_interest、A 是 avoid，也应判断为 rejected（主要主题权重更高）",
		"     - 例 2：全局配置主题 X 为 interest，来源配置主题 Y 为 high_interest → 文章同时讲 X 和 Y → 两个配置都在起作用，都是高兴趣，应判断为 critical 或 recommended（根据主题占比判断）",
		"     - 例 3：全局配置主题 X 为 high_interest，来源配置主题 Y 为 uninterested → 文章同时讲 X 和 Y，两者比重相当 → 综合评定可能为 recommended 或 optional（X 加分，Y 减分，需根据主题占比和配置综合权衡，权衡时来源配置的权重更高）",
		"",
		"**重要**：",
		"- 禁止编写脚本、禁止匹配关键词，要理解标题和摘要实际在讲什么",
		"- 直接根据理解判断分级",
		"",
		"## 输出",
		"",
		"逐条给出所有 item 的最终分级结果。",
		"每个 item 都必须返回 guid、level、reason。",
		"完成判断后，立即调用 `write_grade_results` 工具写入完整结果。",
		"",
		fmt.Sprintf("Global high interest: %s", req.GlobalHigh),
		fmt.Sprintf("Global interest: %s", req.GlobalInterest),
		fmt.Sprintf("Global uninterested: %s", req.GlobalUninterested),
		fmt.Sprintf("Global avoid: %s", req.GlobalAvoid),
		fmt.Sprintf("Source high interest: %s", req.High),
		fmt.Sprintf("Source interest: %s", req.Interest),
		fmt.Sprintf("Source uninterested: %s", req.Uninterested),
		fmt.Sprintf("Source avoid: %s", req.Avoid),
		fmt.Sprintf("Source context: %s", req.SourceContext),
		fmt.Sprintf("Extra context: %s", req.Context),
	)
	for _, item := range req.Items {
		lines = append(lines, fmt.Sprintf("Item guid=%s title=%q meta=%q", item.GUID, item.Title, item.Meta))
	}
	return strings.Join(lines, "\n")
}

func buildSummaryPrompt(req SummaryRequest) string {
	extra, _ := json.Marshal(req.Extra)
	return strings.Join([]string{
		"# 角色",
		"",
		"你是一个专业的内容总结专家，你的职责是为用户产出易于阅读的 RSS 标题和内容总结，少数情况下补充扩展阅读内容。",
		"",
		"你必须调用 `write_summary` 工具完成输出。",
		"禁止输出普通文本、禁止输出 JSON、禁止解释过程。",
		"只允许调用一次工具。",
		`工具参数必须是：{"guid":"string","title":"string","description":"string","rejected":true|false}`,
		"guid 字段必须原样复制输入里的 GUID，禁止翻译、改写、截断或重新生成。",
		"",
		"# 任务",
		"",
		"深度理解一篇文章的内容，生成首选语言的标题和总结。",
		"",
		"## 理解内容",
		"",
		"- 标题",
		"- 描述",
		"- extra 中的所有额外数据（正文、评论等）",
		"- 如果 extra.content 包含 [IMAGE_N] 占位符，说明文中有图片，位置信息已标记",
		"- 如果有 extra.images 数组，包含图片的元数据（src、alt、尺寸等）",
		"",
		"## 输出要求",
		"",
		"- 所有输出使用首选语言",
		"- 专有名词或缩写首次出现时，用括号标注原文，如：「错误检测与纠正（EDAC）」",
		"",
		"## 生成标题",
		"",
		"- 清晰表达内容主题",
		"- 如果原标题模糊需根据实际内容重新生成",
		"",
		"## 生成总结",
		"",
		"总结包含正文和可选的「你知道吗？」章节。",
		"",
		"**通用要求**：",
		"- 使用 HTML 格式",
		"- 风格要求：报道风格，客观准确，平实陈述，简洁凝练，自然流畅，易于阅读，避免冗余和夸张",
		"",
		"**正文**：",
		"- 总结文章的主要内容、重要信息和主要观点以及社区反馈等，以简化理解为目的，注意不是原样翻译",
		"- 保留原文中重要内容的超链接（使用 <a href=\"...\"> 标签）",
		"- 如果 extra.images 存在，根据原文中 [IMAGE_N] 的位置和描述，猜测其作用，在总结中适当位置插入对应的 <img> 标签（禁止试图下载图片）",
		"- 图片标签格式：<img src=\"...\" alt=\"...\" />（保留 width/height 属性如果有）",
		"",
		"**「你知道吗？」章节（可选）**：",
		"- 提供原文内容以外的扩展阅读",
		"- 仅在需要解释文章中冷门小众的专业概念、知识，或有必要补充额外视角时，才需要在正文后添加此章节作为扩展补充，大多数文章不需要此章节",
		"- 假设读者为相关领域的平均水平，无需解释领域常见概念和基础知识",
		"",
		"## rejected 字段",
		"",
		"如果文章内容（非标题）本身表明该条目实际上不值得阅读（如内容为空、严重误导、或与标题描述完全不符），将 rejected 设为 true。正常情况下设为 false。",
		"",
		"## 输出",
		"",
		"必须返回 guid、title、description、rejected。",
		"完成判断后，立即调用 `write_summary` 工具写入结果。",
		"",
		fmt.Sprintf("Preferred language: %s", req.PreferredLanguage),
		fmt.Sprintf("Source context: %s", req.SourceContext),
		fmt.Sprintf("Extra context: %s", req.Context),
		fmt.Sprintf("GUID: %s", req.GUID),
		fmt.Sprintf("Title: %s", req.Title),
		fmt.Sprintf("Description: %s", req.Description),
		fmt.Sprintf("Extra: %s", string(extra)),
	}, "\n")
}
