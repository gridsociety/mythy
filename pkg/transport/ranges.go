package transport

import "sort"

// MergeOptions tunes the bulk-read planner.
//
//   Gap      = number of unrequested registers we'll fill with throwaway
//              reads to merge two adjacent same-FC ranges. Default 0
//              (strict contiguous merge).
//   MaxBatch = hard cap on Count per request. Modbus FC03/FC04 allow up
//              to 125 registers; default 125.
type MergeOptions struct {
	Gap      uint16
	MaxBatch uint16
}

// MergeRanges consolidates a list of RangePlan requests into the minimum
// number of Modbus reads, respecting the gap budget and the per-request
// cap. Different FCs are never merged. Adapted from rospo's
// pkg/protocol/modbus/range.go (algorithm only; copied for project
// independence).
func MergeRanges(in []RangePlan, opts MergeOptions) []RangePlan {
	if len(in) == 0 {
		return nil
	}
	if opts.MaxBatch == 0 {
		opts.MaxBatch = 125
	}

	cp := make([]RangePlan, len(in))
	copy(cp, in)
	sort.Slice(cp, func(i, j int) bool {
		if cp[i].FC != cp[j].FC {
			return cp[i].FC < cp[j].FC
		}
		return cp[i].StartAddr < cp[j].StartAddr
	})

	var out []RangePlan
	for _, r := range cp {
		if len(out) == 0 {
			out = append(out, r)
			continue
		}
		last := &out[len(out)-1]
		if last.FC != r.FC {
			out = append(out, r)
			continue
		}
		gap := int(r.StartAddr) - last.End()
		if gap < 0 {
			// Overlap — extend if needed.
			newEnd := r.End()
			if newEnd > last.End() {
				if newCount := newEnd - int(last.StartAddr); newCount <= int(opts.MaxBatch) {
					last.Count = uint16(newCount)
				} else {
					out = append(out, r)
				}
			}
			continue
		}
		if gap > int(opts.Gap) {
			out = append(out, r)
			continue
		}
		newCount := r.End() - int(last.StartAddr)
		if newCount > int(opts.MaxBatch) {
			out = append(out, r)
			continue
		}
		last.Count = uint16(newCount)
	}
	return out
}
