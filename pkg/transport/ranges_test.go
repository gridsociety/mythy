package transport

import (
	"reflect"
	"testing"
)

func TestMergeRangesEmpty(t *testing.T) {
	out := MergeRanges(nil, MergeOptions{})
	if len(out) != 0 {
		t.Errorf("empty input → empty output, got %v", out)
	}
}

func TestMergeRangesContiguous(t *testing.T) {
	in := []RangePlan{
		{FC: 4, StartAddr: 100, Count: 1},
		{FC: 4, StartAddr: 101, Count: 1},
		{FC: 4, StartAddr: 102, Count: 2}, // ends at 104
	}
	out := MergeRanges(in, MergeOptions{})
	want := []RangePlan{{FC: 4, StartAddr: 100, Count: 4}}
	if !reflect.DeepEqual(out, want) {
		t.Errorf("got %+v, want %+v", out, want)
	}
}

func TestMergeRangesGapStrictDefault(t *testing.T) {
	in := []RangePlan{
		{FC: 4, StartAddr: 100, Count: 1},
		{FC: 4, StartAddr: 102, Count: 1}, // 1-reg gap
	}
	out := MergeRanges(in, MergeOptions{})
	want := []RangePlan{
		{FC: 4, StartAddr: 100, Count: 1},
		{FC: 4, StartAddr: 102, Count: 1},
	}
	if !reflect.DeepEqual(out, want) {
		t.Errorf("strict mode must NOT merge across gaps; got %+v", out)
	}
}

func TestMergeRangesGapBudget(t *testing.T) {
	in := []RangePlan{
		{FC: 4, StartAddr: 100, Count: 1},
		{FC: 4, StartAddr: 102, Count: 1},
	}
	out := MergeRanges(in, MergeOptions{Gap: 1})
	want := []RangePlan{{FC: 4, StartAddr: 100, Count: 3}}
	if !reflect.DeepEqual(out, want) {
		t.Errorf("gap=1 must merge across 1-reg gap; got %+v", out)
	}
}

func TestMergeRangesMaxBatch(t *testing.T) {
	in := []RangePlan{
		{FC: 4, StartAddr: 100, Count: 100},
		{FC: 4, StartAddr: 200, Count: 100},
	}
	out := MergeRanges(in, MergeOptions{Gap: 200, MaxBatch: 125})
	if len(out) != 2 {
		t.Errorf("expected 2 batches because MaxBatch=125 < 200; got %d: %+v", len(out), out)
	}
	for _, r := range out {
		if int(r.Count) > 125 {
			t.Errorf("batch exceeded MaxBatch: %+v", r)
		}
	}
}

func TestMergeRangesMixedFC(t *testing.T) {
	in := []RangePlan{
		{FC: 3, StartAddr: 100, Count: 1},
		{FC: 4, StartAddr: 101, Count: 1},
	}
	out := MergeRanges(in, MergeOptions{})
	if len(out) != 2 {
		t.Errorf("different FCs must NOT merge; got %+v", out)
	}
}
