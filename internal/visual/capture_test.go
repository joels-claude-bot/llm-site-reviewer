package visual

import (
	"reflect"
	"testing"
)

func TestScrollPositions(t *testing.T) {
	for _, tt := range []struct {
		page, viewport int
		want           []int
	}{
		{3100, 900, []int{0, 630, 1260, 1890, 2200}},
		{2792, 900, []int{0, 630, 1260, 1892}},
		{600, 900, []int{0}},
	} {
		got, err := ScrollPositions(tt.page, tt.viewport, .7)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ScrollPositions(%d, %d) = %v, want %v", tt.page, tt.viewport, got, tt.want)
		}
	}
}
