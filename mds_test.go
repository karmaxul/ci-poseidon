// mds_test.go — Tests for MDS matrix construction
//
// Tests verify:
//   1. Both circulant and ci-derived matrices satisfy the MDS property
//   2. Matrix application is correct (dimensions, field membership)
//   3. Different inputs produce different outputs
//   4. tHz + nm = 10800 invariant holds across seed entries
//   5. Baseline and ci-derived produce different matrices (control vs experimental)
//   6. All supported widths (t=2,3,4,6) pass for both construction methods

package ciposeidon

import (
	"math/big"
	"testing"
)

var supportedWidths = []int{2, 3, 4, 6}

// ── MDS property tests ────────────────────────────────────────────────────────

func TestCirculantMDSProperty(t *testing.T) {
	p := bn254()
	for _, w := range supportedWidths {
		m := NewCirculantMDS(w, p)
		if !m.IsMDS() {
			t.Errorf("circulant t=%d: failed MDS property", w)
		}
	}
}

func TestCiDerivedMDSProperty(t *testing.T) {
	p := bn254()
	for _, w := range supportedWidths {
		m := NewCiDerivedMDS(w, p)
		if !m.IsMDS() {
			t.Errorf("ci-derived t=%d: failed MDS property", w)
		}
	}
}

func TestCirculantMDSPropertyBLS(t *testing.T) {
	p := bls()
	for _, w := range supportedWidths {
		m := NewCirculantMDS(w, p)
		if !m.IsMDS() {
			t.Errorf("circulant BLS t=%d: failed MDS property", w)
		}
	}
}

func TestCiDerivedMDSPropertyBLS(t *testing.T) {
	p := bls()
	for _, w := range supportedWidths {
		m := NewCiDerivedMDS(w, p)
		if !m.IsMDS() {
			t.Errorf("ci-derived BLS t=%d: failed MDS property", w)
		}
	}
}

// ── Matrix structure tests ────────────────────────────────────────────────────

func TestMatrixDimensions(t *testing.T) {
	p := bn254()
	for _, w := range supportedWidths {
		for _, label := range []string{"circulant", "ci-derived"} {
			var m *MDSMatrix
			if label == "circulant" {
				m = NewCirculantMDS(w, p)
			} else {
				m = NewCiDerivedMDS(w, p)
			}
			if len(m.Entries) != w {
				t.Errorf("%s t=%d: expected %d rows, got %d", label, w, w, len(m.Entries))
			}
			for i, row := range m.Entries {
				if len(row) != w {
					t.Errorf("%s t=%d row %d: expected %d cols, got %d", label, w, i, w, len(row))
				}
			}
		}
	}
}

func TestMatrixEntriesInField(t *testing.T) {
	p := bn254()
	for _, w := range supportedWidths {
		for _, label := range []string{"circulant", "ci-derived"} {
			var m *MDSMatrix
			if label == "circulant" {
				m = NewCirculantMDS(w, p)
			} else {
				m = NewCiDerivedMDS(w, p)
			}
			for i, row := range m.Entries {
				for j, v := range row {
					mustInField(t, label+" entry["+string(rune('0'+i))+"]["+string(rune('0'+j))+"]", v, p)
					mustPositive(t, label+" entry", v)
				}
			}
		}
	}
}

func TestMatrixNoZeroEntries(t *testing.T) {
	p := bn254()
	zero := bigZero()
	for _, w := range supportedWidths {
		for _, label := range []string{"circulant", "ci-derived"} {
			var m *MDSMatrix
			if label == "circulant" {
				m = NewCirculantMDS(w, p)
			} else {
				m = NewCiDerivedMDS(w, p)
			}
			for i, row := range m.Entries {
				for j, v := range row {
					if v.Cmp(zero) == 0 {
						t.Errorf("%s t=%d: zero entry at [%d][%d]", label, w, i, j)
					}
				}
			}
		}
	}
}

// ── Apply tests ───────────────────────────────────────────────────────────────

func TestMatrixApplyDimensions(t *testing.T) {
	p := bn254()
	for _, w := range supportedWidths {
		m := NewCirculantMDS(w, p)
		state := makeState(w, p)
		out := m.Apply(state)
		if len(out) != w {
			t.Errorf("Apply t=%d: expected output width %d, got %d", w, w, len(out))
		}
	}
}

func TestMatrixApplyOutputInField(t *testing.T) {
	p := bn254()
	for _, w := range supportedWidths {
		for _, label := range []string{"circulant", "ci-derived"} {
			var m *MDSMatrix
			if label == "circulant" {
				m = NewCirculantMDS(w, p)
			} else {
				m = NewCiDerivedMDS(w, p)
			}
			state := makeState(w, p)
			out := m.Apply(state)
			for i, v := range out {
				mustInField(t, label+" apply out", v, p)
				_ = i
			}
		}
	}
}

func TestMatrixApplyDifferentInputsDifferentOutputs(t *testing.T) {
	p := bn254()
	for _, w := range supportedWidths {
		m := NewCirculantMDS(w, p)
		s1 := makeState(w, p)
		s2 := makeStateOffset(w, p, 100)
		o1 := m.Apply(s1)
		o2 := m.Apply(s2)
		allSame := true
		for i := range o1 {
			if o1[i].Cmp(o2[i]) != 0 {
				allSame = false
				break
			}
		}
		if allSame {
			t.Errorf("circulant t=%d: different inputs produced same output", w)
		}
	}
}

func TestMatrixApplyDeterminism(t *testing.T) {
	p := bn254()
	for _, w := range supportedWidths {
		m := NewCiDerivedMDS(w, p)
		s := makeState(w, p)
		o1 := m.Apply(s)
		o2 := m.Apply(s)
		for i := range o1 {
			if o1[i].Cmp(o2[i]) != 0 {
				t.Errorf("ci-derived t=%d: Apply not deterministic at index %d", w, i)
			}
		}
	}
}

// ── Harmonic invariant test ───────────────────────────────────────────────────

// TestTHzNmComplementarity verifies the core harmonic property of the resonance
// matrix: for every seed entry, tHz×10 + nm×10 = 10800 exactly.
// This is the built-in balance that distinguishes ci-derived from arbitrary matrices.
func TestTHzNmComplementarity(t *testing.T) {
	for width, seeds := range seedEntries {
		for i, s := range seeds {
			sum := s.tHz + s.nm
			if sum != 1080.0 {
				t.Errorf("width %d seed %d: tHz(%.1f) + nm(%.1f) = %.1f, expected 1080",
					width, i, s.tHz, s.nm, sum)
			}
		}
	}
}

// ── Baseline vs ci-derived distinction ───────────────────────────────────────

func TestBaselineAndCiDerivedDiffer(t *testing.T) {
	p := bn254()
	for _, w := range supportedWidths {
		base := NewCirculantMDS(w, p)
		ciDer := NewCiDerivedMDS(w, p)
		allSame := true
		for i := 0; i < w; i++ {
			for j := 0; j < w; j++ {
				if base.Entries[i][j].Cmp(ciDer.Entries[i][j]) != 0 {
					allSame = false
					break
				}
			}
		}
		if allSame {
			t.Errorf("t=%d: circulant and ci-derived matrices are identical — no experiment possible", w)
		}
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func bigZero() *big.Int { return big.NewInt(0) }

func makeState(width int, p *big.Int) []*big.Int {
	s := make([]*big.Int, width)
	for i := range s {
		s[i] = KConstantField(i, p)
	}
	return s
}

func makeStateOffset(width int, p *big.Int, offset int) []*big.Int {
	s := make([]*big.Int, width)
	for i := range s {
		s[i] = KConstantField(i+offset, p)
	}
	return s
}
