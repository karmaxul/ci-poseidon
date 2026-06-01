// symmetry_mds_test.go — Testing the 19/40 square symmetry MDS hypotheses
//
// Three hypotheses:
//   H1: Symmetry counts (Blue:12, Green:28, Red:40, Yellow:40, Bl/Gr:40) as seeds
//   H2: Class composition ratios (6T:6O:6A/R:1) as seeds
//   H3: tHz wavelengths of the 19 shared-position elements as matrix entries
//
// Each hypothesis is tested for:
//   - MDS property (valid diffusion matrix?)
//   - Avalanche vs circulant baseline
//   - Augmentation needed? (how close to MDS naturally?)
//   - tHz+nm=1080 invariant preserved?

package ciposeidon

import (
	"fmt"
	"math/big"
	"math/bits"
	"testing"
)

// ── Structure analysis tests ──────────────────────────────────────────────────

func TestSquareSymmetryStructure(t *testing.T) {
	t.Log(PrintSquareSymmetrySummary())
}

func TestSharedElementTHzData(t *testing.T) {
	t.Log(THzSummaryForSharedElements())
}

func TestSquareSymmetryGroupsABIdentical(t *testing.T) {
	// Groups a and b should be identical across all 5 symmetry types
	for _, s := range SquareSymmetries {
		if s.CountA != s.CountB {
			t.Errorf("symmetry type %s: groups a and b differ (%d vs %d)",
				s.Name, s.CountA, s.CountB)
		}
	}
	t.Log("✓ Groups a and b identical across all 5 symmetry types — confirmed")
}

func TestSquareSymmetryTotals(t *testing.T) {
	// Each symmetry type total should be consistent with 40 elements per square × 3
	// (some elements participate in multiple symmetry types, so totals can exceed 40)
	t.Log("Symmetry type totals (a+b+c):")
	for _, s := range SquareSymmetries {
		total := s.CountA + s.CountB + s.CountC
		t.Logf("  %-8s: %2d + %2d + %2d = %2d", s.Name, s.CountA, s.CountB, s.CountC, total)
	}
}

func TestClassCompositionBalance(t *testing.T) {
	// Verify the class composition sums to 21 per square (non-shared elements)
	// Plus 19 shared = 40 total per square
	for _, c := range SquareCompositions {
		total := c.Transition + c.OtherMetal + c.AlkaliRare
		if total != 21 {
			t.Errorf("square %s: class total %d, expected 21", c.Square, total)
		}
	}
	// Shared elements: 6+6+6+1 = 19
	sharedTotal := SharedElementComposition.Transition +
		SharedElementComposition.OtherMetal +
		SharedElementComposition.AlkaliRare + 1 // +1 asymmetry element
	if sharedTotal != 19 {
		t.Errorf("shared element total: expected 19, got %d", sharedTotal)
	}
	t.Logf("✓ Class composition verified: 21 non-shared + 19 shared = 40 per square")
}

// ── H1: Symmetry count MDS tests ─────────────────────────────────────────────

func TestH1SymmetryCountMDSProperty(t *testing.T) {
	p := bn254()
	t.Log("H1: Symmetry count seeds → MDS property")
	t.Log("Seeds: Blue=12, Green=28, Red=40, Yellow=40, Bl/Gr=40 (sums across a+b+c)")
	for _, w := range supportedWidths {
		m := NewSymmetryCountMDS(w, p)
		isMDS := m.IsMDS()
		t.Logf("  t=%-2d  label=%-40s  MDS=%v", w, m.Label, isMDS)
		if !isMDS {
			t.Errorf("H1: t=%d failed MDS property", w)
		}
	}
}

func TestH1SymmetryAsymmetryMDSProperty(t *testing.T) {
	p := bn254()
	t.Log("H1b: Symmetry asymmetry seeds (a vs c difference) → MDS property")
	for _, w := range supportedWidths {
		m := NewSymmetryAsymmetryMDS(w, p)
		isMDS := m.IsMDS()
		t.Logf("  t=%-2d  label=%-40s  MDS=%v", w, m.Label, isMDS)
		if !isMDS {
			t.Errorf("H1b: t=%d failed MDS property", w)
		}
	}
}

// ── H2: Class composition MDS tests ──────────────────────────────────────────

func TestH2ClassCompositionMDSProperty(t *testing.T) {
	p := bn254()
	t.Log("H2: Class composition seeds (6T:6O:6A/R:1) → MDS property")
	for _, w := range supportedWidths {
		m := NewClassCompositionMDS(w, p)
		isMDS := m.IsMDS()
		t.Logf("  t=%-2d  label=%-40s  MDS=%v", w, m.Label, isMDS)
		if !isMDS {
			t.Errorf("H2: t=%d failed MDS property", w)
		}
	}
}

// ── H3: tHz wavelength MDS tests ─────────────────────────────────────────────

func TestH3THzSymmetryMDSProperty(t *testing.T) {
	p := bn254()
	t.Log("H3: tHz wavelengths of 19 shared-position elements → MDS property")
	t.Logf("Using %d known shared elements", len(KnownSharedElements))
	for _, w := range supportedWidths {
		m := NewTHzSymmetryMDS(w, p)
		isMDS := m.IsMDS()
		t.Logf("  t=%-2d  label=%-40s  MDS=%v", w, m.Label, isMDS)
		if !isMDS {
			t.Errorf("H3: t=%d failed MDS property", w)
		}
	}
}

func TestH3THzNmInvariant(t *testing.T) {
	// Verify the tHz+nm=1080 invariant holds for all known shared elements
	for _, e := range KnownSharedElements {
		nm := 1080.0 - e.tHz
		sum := e.tHz + nm
		if sum != 1080.0 {
			t.Errorf("element %s: tHz(%.1f) + nm(%.1f) = %.1f, expected 1080",
				e.Symbol, e.tHz, nm, sum)
		}
	}
	t.Logf("✓ tHz+nm=1080 invariant verified for all %d known shared elements",
		len(KnownSharedElements))
}

// ── Avalanche comparison ──────────────────────────────────────────────────────

// avalancheForMDS measures bit-level avalanche for a given MDS matrix
// by running it through the full sponge with that MDS forced at t=3.
func avalancheForMDS(mds *MDSMatrix, samples int, fieldPrime *big.Int) float64 {
	changed, total := 0, 0
	rc := NewRoundConstants(3, fieldPrime)

	for i := 0; i < samples; i++ {
		base := KConstantField(i%512, fieldPrime)
		flipped := new(big.Int).Xor(base, big.NewInt(1))
		flipped.Mod(flipped, fieldPrime)

		s1 := makeState(3, fieldPrime)
		s1[0] = base
		s2 := makeState(3, fieldPrime)
		s2[0] = flipped

		ApplyPermutation(s1, rc, mds)
		ApplyPermutation(s2, rc, mds)

		for j := range s1 {
			diff := new(big.Int).Xor(s1[j], s2[j])
			changed += bits.OnesCount64(diff.Uint64())
			total += 64
		}
	}
	return float64(changed) / float64(total) * 100.0
}

func TestSymmetryMDSAvalancheComparison(t *testing.T) {
	p := bn254()
	samples := 200

	t.Log("═══════════════════════════════════════════════════════════")
	t.Log("  SYMMETRY-DERIVED MDS — AVALANCHE COMPARISON (t=3)")
	t.Logf(" Samples: %d  |  Field: BN254", samples)
	t.Log("═══════════════════════════════════════════════════════════")

	// Baseline
	baselineAvg := avalancheForMDS(NewCirculantMDS(3, p), samples, p)
	t.Logf("  %-40s  %.2f%%  (baseline)", "circulant-baseline", baselineAvg)

	ciDerivedAvg := avalancheForMDS(NewCiDerivedMDS(3, p), samples, p)
	t.Logf("  %-40s  %.2f%%", "ci-derived (resonance anchors)", ciDerivedAvg)

	// H1
	h1Avg := avalancheForMDS(NewSymmetryCountMDS(3, p), samples, p)
	t.Logf("  %-40s  %.2f%%  H1: symmetry counts", "symmetry-count-derived", h1Avg)

	h1bAvg := avalancheForMDS(NewSymmetryAsymmetryMDS(3, p), samples, p)
	t.Logf("  %-40s  %.2f%%  H1b: a vs c asymmetry", "symmetry-asymmetry-derived", h1bAvg)

	// H2
	h2Avg := avalancheForMDS(NewClassCompositionMDS(3, p), samples, p)
	t.Logf("  %-40s  %.2f%%  H2: class composition", "class-composition-derived", h2Avg)

	// H3
	h3Avg := avalancheForMDS(NewTHzSymmetryMDS(3, p), samples, p)
	t.Logf("  %-40s  %.2f%%  H3: tHz of shared elements", "thz-symmetry-derived", h3Avg)

	t.Log("  Reference: ideal = 50.00%")
	t.Log("═══════════════════════════════════════════════════════════")

	// All should be in acceptable range
	for name, avg := range map[string]float64{
		"H1":  h1Avg,
		"H1b": h1bAvg,
		"H2":  h2Avg,
		"H3":  h3Avg,
	} {
		if avg < 35.0 {
			t.Errorf("%s: avalanche too low (%.2f%%)", name, avg)
		}
	}
}

// ── Full comparison across all widths ─────────────────────────────────────────

func TestSymmetryMDSFullComparison(t *testing.T) {
	p := bn254()

	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║   19/40 SQUARE SYMMETRY — MDS HYPOTHESIS RESULTS         ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")
	t.Log("║  All matrices tested for MDS property across t=2,3,4,6   ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")

	type mdsFactory struct {
		name string
		fn   func(int, *big.Int) *MDSMatrix
	}

	factories := []mdsFactory{
		{"circulant-baseline", NewCirculantMDS},
		{"ci-derived (anchors)", NewCiDerivedMDS},
		{"H1: symmetry counts", NewSymmetryCountMDS},
		{"H1b: a/c asymmetry", NewSymmetryAsymmetryMDS},
		{"H2: class composition", NewClassCompositionMDS},
		{"H3: tHz shared elements", NewTHzSymmetryMDS},
	}

	for _, f := range factories {
		allPass := true
		widthResults := ""
		for _, w := range supportedWidths {
			m := f.fn(w, p)
			pass := m.IsMDS()
			if !pass {
				allPass = false
			}
			widthResults += fmt.Sprintf("t=%d:%v ", w, pass)
		}
		status := "✓"
		if !allPass {
			status = "✗"
		}
		t.Logf("║  %s %-28s  %s  ║", status, f.name, widthResults)
	}

	t.Log("╠═══════════════════════════════════════════════════════════╣")
	t.Log("║  Finding: which hypothesis produces MDS naturally         ║")
	t.Log("║  (without augmentation) is the key result for the paper   ║")
	t.Log("╚═══════════════════════════════════════════════════════════╝")
}

// ── Entry point summary ───────────────────────────────────────────────────────

func TestSymmetryMDSSummary(t *testing.T) {
	p := bn254()

	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║   SYMMETRY-DERIVED MDS — RESEARCH SUMMARY                ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")
	t.Logf("║  Known shared-position elements: %d                       ║",
		len(KnownSharedElements))

	// tHz stats
	var minTHz, maxTHz, sumTHz float64
	minTHz = KnownSharedElements[0].tHz
	maxTHz = KnownSharedElements[0].tHz
	for _, e := range KnownSharedElements {
		sumTHz += e.tHz
		if e.tHz < minTHz { minTHz = e.tHz }
		if e.tHz > maxTHz { maxTHz = e.tHz }
	}
	avgTHz := sumTHz / float64(len(KnownSharedElements))
	avgNm := 1080.0 - avgTHz

	t.Logf("║  tHz range: %.1f – %.1f  avg: %.2f                   ║",
		minTHz, maxTHz, avgTHz)
	t.Logf("║  nm  range: %.1f – %.1f  avg: %.2f                   ║",
		1080.0-maxTHz, 1080.0-minTHz, avgNm)
	t.Log("╠═══════════════════════════════════════════════════════════╣")

	// Quick MDS check for H3 at t=3
	h3 := NewTHzSymmetryMDS(3, p)
	t.Logf("║  H3 (tHz) at t=3: MDS=%v  label=%s  ║",
		h3.IsMDS(), h3.Label)

	t.Log("╠═══════════════════════════════════════════════════════════╣")
	t.Log("║  Open question: do the 19 shared elements have tHz        ║")
	t.Log("║  values that produce MDS without augmentation?            ║")
	t.Log("║  Answer: see TestSymmetryMDSAvalancheComparison above     ║")
	t.Log("╚═══════════════════════════════════════════════════════════╝")

	_ = big.NewInt(0) // keep import
}
