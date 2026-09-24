package config

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestDefaults(t *testing.T) {
	d := Defaults()
	if !d.Enabled {
		t.Error("Enabled should default to true")
	}
	if d.Events.Ask.DurationSeconds != 20 {
		t.Errorf("Events.Ask.DurationSeconds = %v, want 20", d.Events.Ask.DurationSeconds)
	}
	if d.Events.Done.MinTurnSeconds != 0 {
		t.Errorf("Events.Done.MinTurnSeconds = %v, want 0", d.Events.Done.MinTurnSeconds)
	}
	if d.Display.Corner != "top-right" {
		t.Errorf("Display.Corner = %q, want top-right", d.Display.Corner)
	}
	if d.Sound.Volume != 0.6 {
		t.Errorf("Sound.Volume = %v, want 0.6", d.Sound.Volume)
	}
	if d.Labels.FocusButton != "Back to {app}" {
		t.Errorf("Labels.FocusButton = %q", d.Labels.FocusButton)
	}
	if d.ContextWindow.Default != 200000 {
		t.Errorf("ContextWindow.Default = %v, want 200000", d.ContextWindow.Default)
	}
	if d.ContextWindow.Models["opus"] != 1000000 {
		t.Errorf("ContextWindow.Models[opus] = %v, want 1000000", d.ContextWindow.Models["opus"])
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]interface{}
		want func(Config) bool
	}{
		{
			name: "empty override keeps defaults",
			raw:  map[string]interface{}{},
			want: func(c Config) bool { return c.Enabled && c.Display.Corner == "top-right" },
		},
		{
			name: "top level override",
			raw:  map[string]interface{}{"enabled": false},
			want: func(c Config) bool { return !c.Enabled && c.Display.Corner == "top-right" },
		},
		{
			name: "nested override keeps siblings",
			raw: map[string]interface{}{
				"events": map[string]interface{}{
					"ask": map[string]interface{}{"title": "custom"},
				},
			},
			want: func(c Config) bool {
				return c.Events.Ask.Title == "custom" && c.Events.Ask.Body == "{question}" && c.Events.Done.Title != ""
			},
		},
		{
			name: "map override adds a model without dropping defaults",
			raw: map[string]interface{}{
				"contextWindow": map[string]interface{}{
					"models": map[string]interface{}{"haiku": float64(500000)},
				},
			},
			want: func(c Config) bool {
				return c.ContextWindow.Models["haiku"] == 500000 && c.ContextWindow.Models["opus"] == 1000000
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Merge(tt.raw)
			if !tt.want(got) {
				t.Errorf("Merge(%v) = %+v, unexpected", tt.raw, got)
			}
		})
	}
}

func TestLoadRawMissingFile(t *testing.T) {
	raw, err := LoadRaw(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("LoadRaw() error = %v", err)
	}
	if len(raw) != 0 {
		t.Errorf("LoadRaw() = %v, want empty", raw)
	}
}

func TestLoadRawInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRaw(path); err == nil {
		t.Error("LoadRaw() expected an error for invalid JSON")
	}
}

func TestSaveRawRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.json")
	raw := map[string]interface{}{"enabled": false}
	if err := SaveRaw(path, raw); err != nil {
		t.Fatalf("SaveRaw() error = %v", err)
	}
	got, err := LoadRaw(path)
	if err != nil {
		t.Fatalf("LoadRaw() error = %v", err)
	}
	if !reflect.DeepEqual(got, raw) {
		t.Errorf("LoadRaw() = %v, want %v", got, raw)
	}
}

func TestSaveRawWritesPrivateFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes do not apply on windows")
	}
	path := filepath.Join(t.TempDir(), "sub", "config.json")
	if err := SaveRaw(path, map[string]interface{}{"enabled": false}); err != nil {
		t.Fatalf("SaveRaw() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("config file mode = %o, want %o", got, 0o600)
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Errorf("config dir mode = %o, want %o", got, 0o700)
	}
}

func TestGet(t *testing.T) {
	cfg := Defaults()
	tests := []struct {
		path    string
		want    interface{}
		wantErr bool
	}{
		{"enabled", true, false},
		{"display.corner", "top-right", false},
		{"events.ask.durationSeconds", 20.0, false},
		{"contextWindow.default", 200000, false},
		{"contextWindow.models.opus", 1000000, false},
		{"nope", nil, true},
		{"display.nope", nil, true},
		{"enabled.extra", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := Get(cfg, tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Get(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get(%q) = %v (%T), want %v (%T)", tt.path, got, got, tt.want, tt.want)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		in   interface{}
		want string
	}{
		{"hi", "hi"},
		{true, "true"},
		{42, "42"},
		{0.6, "0.6"},
	}
	for _, tt := range tests {
		if got := FormatValue(tt.in); got != tt.want {
			t.Errorf("FormatValue(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSet(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		value   string
		wantErr bool
		check   func(map[string]interface{}) bool
	}{
		{
			name: "bool", path: "enabled", value: "false",
			check: func(m map[string]interface{}) bool { return m["enabled"] == false },
		},
		{
			name: "number", path: "sound.volume", value: "0.8",
			check: func(m map[string]interface{}) bool {
				sound, _ := m["sound"].(map[string]interface{})
				return sound["volume"] == 0.8
			},
		},
		{
			name: "string", path: "display.corner", value: "bottom-left",
			check: func(m map[string]interface{}) bool {
				d, _ := m["display"].(map[string]interface{})
				return d["corner"] == "bottom-left"
			},
		},
		{
			name: "map key", path: "contextWindow.models.haiku", value: "500000",
			check: func(m map[string]interface{}) bool {
				cw, _ := m["contextWindow"].(map[string]interface{})
				models, _ := cw["models"].(map[string]interface{})
				return models["haiku"] == float64(500000)
			},
		},
		{name: "unknown key", path: "nope", value: "x", wantErr: true},
		{name: "bad bool", path: "enabled", value: "maybe", wantErr: true},
		{name: "bad number", path: "sound.volume", value: "loud", wantErr: true},
		{name: "section not settable", path: "events.ask", value: "x", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := map[string]interface{}{}
			err := Set(raw, tt.path, tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !tt.check(raw) {
				t.Errorf("Set(%q, %q) produced unexpected tree: %v", tt.path, tt.value, raw)
			}
		})
	}
}

func TestSetValueRules(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		value   string
		wantErr bool
	}{
		{name: "corner valid", path: "display.corner", value: "bottom-right"},
		{name: "corner invalid", path: "display.corner", value: "middle", wantErr: true},
		{name: "screen valid", path: "display.screen", value: "mouse"},
		{name: "screen invalid", path: "display.screen", value: "second", wantErr: true},
		{name: "theme valid", path: "display.theme", value: "dark"},
		{name: "theme invalid", path: "display.theme", value: "blue", wantErr: true},
		{name: "maxCards at min", path: "display.maxCards", value: "1"},
		{name: "maxCards at max", path: "display.maxCards", value: "10"},
		{name: "maxCards below min", path: "display.maxCards", value: "0", wantErr: true},
		{name: "maxCards above max", path: "display.maxCards", value: "11", wantErr: true},
		{name: "volume at min", path: "sound.volume", value: "0"},
		{name: "volume at max", path: "sound.volume", value: "1"},
		{name: "volume below min", path: "sound.volume", value: "-0.1", wantErr: true},
		{name: "volume above max", path: "sound.volume", value: "1.1", wantErr: true},
		{name: "durationSeconds zero is sticky, allowed", path: "events.ask.durationSeconds", value: "0"},
		{name: "durationSeconds negative", path: "events.idle.durationSeconds", value: "-1", wantErr: true},
		{name: "minTurnSeconds zero", path: "events.done.minTurnSeconds", value: "0"},
		{name: "minTurnSeconds negative", path: "events.done.minTurnSeconds", value: "-1", wantErr: true},
		{name: "contextWindow default positive", path: "contextWindow.default", value: "100000"},
		{name: "contextWindow default zero rejected", path: "contextWindow.default", value: "0", wantErr: true},
		{name: "contextWindow model positive", path: "contextWindow.models.haiku", value: "500000"},
		{name: "contextWindow model zero rejected", path: "contextWindow.models.haiku", value: "0", wantErr: true},
		{name: "contextWindow model negative rejected", path: "contextWindow.models.haiku", value: "-1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := map[string]interface{}{}
			err := Set(raw, tt.path, tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Set(%q, %q) error = %v, wantErr %v", tt.path, tt.value, err, tt.wantErr)
			}
			if tt.wantErr && len(raw) != 0 {
				t.Errorf("Set(%q, %q) mutated the tree on error: %v", tt.path, tt.value, raw)
			}
		})
	}
}

func TestSetKeepsUnrelatedKeys(t *testing.T) {
	raw := map[string]interface{}{
		"display": map[string]interface{}{"corner": "bottom-left"},
	}
	if err := Set(raw, "display.theme", "dark"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	d := raw["display"].(map[string]interface{})
	if d["corner"] != "bottom-left" {
		t.Errorf("Set() dropped an unrelated key: %v", d)
	}
	if d["theme"] != "dark" {
		t.Errorf("Set() did not write the new key: %v", d)
	}
}

func TestUnset(t *testing.T) {
	raw := map[string]interface{}{
		"display": map[string]interface{}{"corner": "bottom-left"},
	}
	if err := Unset(raw, "display.corner"); err != nil {
		t.Fatalf("Unset() error = %v", err)
	}
	if _, ok := raw["display"]; ok {
		t.Errorf("Unset() should have pruned the now-empty display map, got %v", raw)
	}
}

func TestUnsetUnknownKey(t *testing.T) {
	if err := Unset(map[string]interface{}{}, "nope"); err == nil {
		t.Error("Unset() expected an error for an unknown key")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]interface{}
		want []string
	}{
		{name: "empty is valid", raw: map[string]interface{}{}, want: nil},
		{
			name: "unknown top level key",
			raw:  map[string]interface{}{"nope": true},
			want: []string{"unknown key: nope"},
		},
		{
			name: "unknown nested key",
			raw: map[string]interface{}{
				"display": map[string]interface{}{"nope": true},
			},
			want: []string{"unknown key: display.nope"},
		},
		{
			name: "bad type",
			raw:  map[string]interface{}{"enabled": "yes"},
			want: []string{"invalid value for enabled: expected a boolean"},
		},
		{
			name: "bad map value",
			raw: map[string]interface{}{
				"contextWindow": map[string]interface{}{
					"models": map[string]interface{}{"opus": "big"},
				},
			},
			want: []string{"invalid value for contextWindow.models.opus: expected a number"},
		},
		{
			name: "section given a scalar",
			raw:  map[string]interface{}{"events": "nope"},
			want: []string{"invalid value for events: expected an object"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Validate(tt.raw)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Validate(%v) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestValidateValueRules(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]interface{}
		want []string
	}{
		{
			name: "corner out of enum",
			raw:  map[string]interface{}{"display": map[string]interface{}{"corner": "middle"}},
			want: []string{"invalid value for display.corner: must be one of top-right, top-left, bottom-right, bottom-left"},
		},
		{
			name: "screen out of enum",
			raw:  map[string]interface{}{"display": map[string]interface{}{"screen": "second"}},
			want: []string{"invalid value for display.screen: must be one of auto, main, mouse"},
		},
		{
			name: "theme out of enum",
			raw:  map[string]interface{}{"display": map[string]interface{}{"theme": "blue"}},
			want: []string{"invalid value for display.theme: must be one of auto, dark, light"},
		},
		{
			name: "maxCards out of range",
			raw:  map[string]interface{}{"display": map[string]interface{}{"maxCards": float64(0)}},
			want: []string{"invalid value for display.maxCards: must be between 1 and 10"},
		},
		{
			name: "volume out of range",
			raw:  map[string]interface{}{"sound": map[string]interface{}{"volume": float64(1.5)}},
			want: []string{"invalid value for sound.volume: must be between 0 and 1"},
		},
		{
			name: "durationSeconds negative",
			raw: map[string]interface{}{
				"events": map[string]interface{}{"ask": map[string]interface{}{"durationSeconds": float64(-1)}},
			},
			want: []string{"invalid value for events.ask.durationSeconds: must be at least 0"},
		},
		{
			name: "minTurnSeconds negative",
			raw: map[string]interface{}{
				"events": map[string]interface{}{"done": map[string]interface{}{"minTurnSeconds": float64(-1)}},
			},
			want: []string{"invalid value for events.done.minTurnSeconds: must be at least 0"},
		},
		{
			name: "contextWindow default not positive",
			raw:  map[string]interface{}{"contextWindow": map[string]interface{}{"default": float64(0)}},
			want: []string{"invalid value for contextWindow.default: must be greater than 0"},
		},
		{
			name: "contextWindow model not positive",
			raw: map[string]interface{}{
				"contextWindow": map[string]interface{}{"models": map[string]interface{}{"haiku": float64(-5)}},
			},
			want: []string{"invalid value for contextWindow.models.haiku: must be greater than 0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Validate(tt.raw)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Validate(%v) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"enabled": false, "display": {"corner": "bottom-left"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Enabled {
		t.Error("Load() did not apply the enabled override")
	}
	if cfg.Display.Corner != "bottom-left" {
		t.Errorf("Display.Corner = %q, want bottom-left", cfg.Display.Corner)
	}
	if cfg.Display.Screen != "auto" {
		t.Errorf("Display.Screen = %q, want default auto", cfg.Display.Screen)
	}
}
