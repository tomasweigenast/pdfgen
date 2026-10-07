package cascade

import (
	"reflect"
	"testing"
)

func TestAddLayerOrdersSublayersBeforeParents(t *testing.T) {
	tests := []struct {
		add  []string
		want []string
	}{
		{[]string{"a", "a.b", "c.d", "a.b.c"}, []string{"a.b.c", "a.b", "a", "c.d", "c"}},
		{[]string{"b", "a", "a.b"}, []string{"b", "a.b", "a"}},
	}
	for _, tt := range tests {
		var layers []string
		for _, l := range tt.add {
			addLayer(l, &layers)
		}
		if !reflect.DeepEqual(layers, tt.want) {
			t.Errorf("adding %q gives %q, want %q", tt.add, layers, tt.want)
		}
	}
}
