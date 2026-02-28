package config

import (
	"testing"
)

func TestMergeStrings(t *testing.T) {
	tests := []struct {
		name     string
		global   string
		specific string
		want     string
	}{
		{
			name:     "both empty",
			global:   "",
			specific: "",
			want:     "",
		},
		{
			name:     "only global",
			global:   "foo,bar",
			specific: "",
			want:     "foo,bar",
		},
		{
			name:     "only specific",
			global:   "",
			specific: "baz,qux",
			want:     "baz,qux",
		},
		{
			name:     "both non-empty",
			global:   "foo,bar",
			specific: "baz,qux",
			want:     "foo,bar,baz,qux",
		},
		{
			name:     "single values both non-empty",
			global:   "foo",
			specific: "bar",
			want:     "foo,bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeStrings(tt.global, tt.specific)
			if got != tt.want {
				t.Errorf("mergeStrings(%q, %q) = %q, want %q", tt.global, tt.specific, got, tt.want)
			}
		})
	}
}

func TestMergeRules(t *testing.T) {
	tests := []struct {
		name     string
		global   string
		specific string
		want     string
	}{
		{
			name:     "empty rules",
			global:   "",
			specific: "",
			want:     "",
		},
		{
			name:     "global only",
			global:   "golang,rust",
			specific: "",
			want:     "golang,rust",
		},
		{
			name:     "specific only",
			global:   "",
			specific: "python,java",
			want:     "python,java",
		},
		{
			name:     "merge both",
			global:   "golang,rust",
			specific: "python,java",
			want:     "golang,rust,python,java",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeRules(tt.global, tt.specific)
			if got != tt.want {
				t.Errorf("MergeRules(%q, %q) = %q, want %q", tt.global, tt.specific, got, tt.want)
			}
		})
	}
}

func TestBuildRulesString(t *testing.T) {
	global := GlobalConfig{
		HighInterest: "go,rust",
		Interest:     "tech",
		Uninterested: "politics",
		Exclude:      "ads",
	}

	tests := []struct {
		name string
		src  SourceConfig
		want string
	}{
		{
			name: "no source overrides",
			src: SourceConfig{
				HighInterest: "",
				Interest:     "",
				Uninterested: "",
				Exclude:      "",
			},
			want: "High: go,rust, Interest: tech, Uninterested: politics, Exclude: ads",
		},
		{
			name: "source overrides high interest",
			src: SourceConfig{
				HighInterest: "ai,ml",
				Interest:     "",
				Uninterested: "",
				Exclude:      "",
			},
			want: "High: go,rust,ai,ml, Interest: tech, Uninterested: politics, Exclude: ads",
		},
		{
			name: "source overrides all",
			src: SourceConfig{
				HighInterest: "ai",
				Interest:     "science",
				Uninterested: "celebrity",
				Exclude:      "spam",
			},
			want: "High: go,rust,ai, Interest: tech,science, Uninterested: politics,celebrity, Exclude: ads,spam",
		},
		{
			name: "source replaces with empty",
			src: SourceConfig{
				HighInterest: "",
				Interest:     "",
				Uninterested: "",
				Exclude:      "",
			},
			want: "High: go,rust, Interest: tech, Uninterested: politics, Exclude: ads",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildRulesString(global, tt.src)
			if got != tt.want {
				t.Errorf("BuildRulesString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveAIConfig(t *testing.T) {
	globalAI := &AIConfig{
		Provider: "gemini",
		Model:    "gemini-2.0-flash",
	}

	tests := []struct {
		name   string
		global *AIConfig
		source *AIConfig
		want   *AIConfig
	}{
		{
			name:   "both nil",
			global: nil,
			source: nil,
			want:   nil,
		},
		{
			name:   "only global",
			global: globalAI,
			source: nil,
			want:   globalAI,
		},
		{
			name:   "only source",
			global: nil,
			source: &AIConfig{Provider: "qwen", Model: "qwen-max"},
			want:   &AIConfig{Provider: "qwen", Model: "qwen-max"},
		},
		{
			name:   "source overrides provider",
			global: globalAI,
			source: &AIConfig{Provider: "qwen", Model: ""},
			want:   &AIConfig{Provider: "qwen", Model: "gemini-2.0-flash"},
		},
		{
			name:   "source overrides model",
			global: globalAI,
			source: &AIConfig{Provider: "", Model: "qwen-max"},
			want:   &AIConfig{Provider: "gemini", Model: "qwen-max"},
		},
		{
			name:   "source overrides both",
			global: globalAI,
			source: &AIConfig{Provider: "qwen", Model: "qwen-max"},
			want:   &AIConfig{Provider: "qwen", Model: "qwen-max"},
		},
		{
			name:   "source empty fields keep global",
			global: globalAI,
			source: &AIConfig{Provider: "", Model: ""},
			want:   globalAI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveAIConfig(tt.global, tt.source)

			if got == nil && tt.want == nil {
				return
			}

			if (got == nil) != (tt.want == nil) {
				t.Fatalf("ResolveAIConfig() returned nil=%v, want nil=%v", got == nil, tt.want == nil)
			}

			if got.Provider != tt.want.Provider {
				t.Errorf("Provider = %q, want %q", got.Provider, tt.want.Provider)
			}
			if got.Model != tt.want.Model {
				t.Errorf("Model = %q, want %q", got.Model, tt.want.Model)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: &Config{
				Sources: []SourceConfig{
					{Name: "test", URL: "https://example.com/rss"},
				},
			},
			wantErr: false,
		},
		{
			name: "no sources",
			cfg: &Config{
				Sources: []SourceConfig{},
			},
			wantErr: true,
			errMsg:  "at least one source is required",
		},
		{
			name: "missing source name",
			cfg: &Config{
				Sources: []SourceConfig{
					{Name: "", URL: "https://example.com/rss"},
				},
			},
			wantErr: true,
			errMsg:  "source[0]: name is required",
		},
		{
			name: "missing source URL",
			cfg: &Config{
				Sources: []SourceConfig{
					{Name: "test", URL: ""},
				},
			},
			wantErr: true,
			errMsg:  "source[0]: URL is required",
		},
		{
			name: "multiple sources with one invalid",
			cfg: &Config{
				Sources: []SourceConfig{
					{Name: "valid", URL: "https://example.com/rss"},
					{Name: "", URL: "https://example2.com/rss"},
				},
			},
			wantErr: true,
			errMsg:  "source[1]: name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error, got nil")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
				return
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %q, want %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}
