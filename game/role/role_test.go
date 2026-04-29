package role

import (
	"server/pkg/pb"
	"testing"
)

func TestComp(t *testing.T) {
	r := &Role{Comps: make([]IComp, pb.TypeComp_TCMax)}
	comp, ok := r.Comps[1].(ICompDataReset)
	if ok {
		t.Log(comp)
	}
}
