package fanoffset

import (
	"testing"

	"github.com/TIANLI0/BS2PRO-Controller/internal/types"
)

func TestMaterializeLearnedCurveAppliesWholeCurve(t *testing.T) {
	c := New(DefaultConfig(), nil)
	curve := []types.FanCurvePoint{
		{Temperature: 50, RPM: 1800},
		{Temperature: 60, RPM: 2300},
		{Temperature: 70, RPM: 2900},
		{Temperature: 90, RPM: 4000},
	}
	signature := c.curveSignature(curve)
	c.learnedMemory.Set(signature, 50, learnedState{offset: 200, confidence: 0.8, hasSeed: true})
	c.learnedMemory.Set(signature, 60, learnedState{offset: -100, confidence: 0.5, hasSeed: true})
	c.learnedMemory.Set(signature, 70, learnedState{offset: 300, confidence: 0.2, hasSeed: true})
	c.learnedMemory.Set(signature, 90, learnedState{offset: 400, confidence: 0.9, hasSeed: true})

	out, changed := c.MaterializeLearnedCurve(curve, 500, 4000)
	if !changed {
		t.Fatal("expected learned offsets to be applied")
	}
	if out[0].Offset != 200 || out[1].Offset != -100 {
		t.Fatalf("whole curve not restored: %+v", out)
	}
	if out[2].Offset != 0 || out[3].Offset != 0 {
		t.Fatalf("weak or high-temp memory should stay off the curve: %+v", out)
	}
	if curve[0].Offset != 0 {
		t.Fatal("input curve was mutated")
	}
}
