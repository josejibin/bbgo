package bitbucket

import "testing"

func TestHasTag(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		tag     string
		want    bool
	}{
		{
			name: "has tag",
			raw:  "Some comment\n<!-- bbgo:tag:ai-review -->",
			tag:  "ai-review",
			want: true,
		},
		{
			name: "no tag",
			raw:  "Some comment without tag",
			tag:  "ai-review",
			want: false,
		},
		{
			name: "different tag",
			raw:  "Some comment\n<!-- bbgo:tag:other -->",
			tag:  "ai-review",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := Comment{Content: Content{Raw: tt.raw}}
			got := HasTag(cm, tt.tag)
			if got != tt.want {
				t.Errorf("HasTag() = %v, want %v", got, tt.want)
			}
		})
	}
}
