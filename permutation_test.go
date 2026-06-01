// permutation_test.go — Tests for the standalone permutation engine
// and Goldilocks field support
//
// Tests verify:
//   1.  Round parameters are correctly tuned per width
//   2.  RoundConstants generates distinct values per width
//   3.  Pow5 S-box correctness
//   4.  ApplyPermutation: determinism, in-field output, avalanche
//   5.  PermuteState convenience wrapper
//   6.  Goldilocks field: constants valid, MDS valid, sponge works
//   7.  Goldilocks avalanche
//   8.  Parameter summary is non-empty
//   9.  Round constants distinct across widths (no cross-contamination)
//   10. ConstantsNeeded matches actual generated constants

package ciposeidon

import (
	"fmt"
	"math/big"
	"math/bits"
	"testing"
)

// ── Round parameter tests ─────────────────────────────────────────────────────

func TestPermutationParamsAllWidths(t *testing.T) {
	expected := map[int][2]int{
		2: {8, 56},
		3: {8, 40},
		4: {8, 32},
		6: {8, 24},
	}
	for width, exp := range expected {
		pp := GetPermutationParams(width)
		if pp.FullRounds != exp[0] {
			t.Errorf("t=%d: FullRounds expected %d, got %d", width, exp[0], pp.FullRounds)
		}
		if pp.PartialRounds != exp[1] {
			t.Errorf("t=%d: PartialRounds expected %d, got %d", width, exp[1], pp.PartialRounds)
		}
	}
}

func TestPermutationParamsTotalRounds(t *testing.T) {
	for _, w := range supportedWidths {
		pp := GetPermutationParams(w)
		if pp.TotalRounds() != pp.FullRounds+pp.PartialRounds {
			t.Errorf("t=%d: TotalRounds mismatch", w)
		}
	}
}

func TestPermutationParamsConstantsNeeded(t *testing.T) {
	for _, w := range supportedWidths {
		pp := GetPermutationParams(w)
		expected := (pp.TotalRounds() + 1) * w
		if pp.ConstantsNeeded() != expected {
			t.Errorf("t=%d: ConstantsNeeded expected %d, got %d", w, expected, pp.ConstantsNeeded())
		}
	}
}

// ── RoundConstants tests ──────────────────────────────────────────────────────

func TestRoundConstantsDimensions(t *testing.T) {
	p := bn254()
	for _, w := range supportedWidths {
		rc := NewRoundConstants(w, p)
		pp := GetPermutationParams(w)
		expectedRounds := pp.TotalRounds() + 1
		if len(rc.Constants) != expectedRounds {
			t.Errorf("t=%d: expected %d rounds of constants, got %d", w, expectedRounds, len(rc.Constants))
		}
		for r, row := range rc.Constants {
			if len(row) != w {
				t.Errorf("t=%d round %d: expected %d constants, got %d", w, r, w, len(row))
			}
		}
	}
}

func TestRoundConstantsInField(t *testing.T) {
	p := bn254()
	for _, w := range supportedWidths {
		rc := NewRoundConstants(w, p)
		for r, row := range rc.Constants {
			for e, v := range row {
				mustInField(t, fmt.Sprintf("RC t=%d r=%d e=%d", w, r, e), v, p)
				mustPositive(t, fmt.Sprintf("RC t=%d r=%d e=%d", w, r, e), v)
			}
		}
	}
}

func TestRoundConstantsDeterminism(t *testing.T) {
	p := bn254()
	for _, w := range supportedWidths {
		rc1 := NewRoundConstants(w, p)
		rc2 := NewRoundConstants(w, p)
		for r := range rc1.Constants {
			for e := range rc1.Constants[r] {
				if rc1.Constants[r][e].Cmp(rc2.Constants[r][e]) != 0 {
					t.Errorf("t=%d: round constants not deterministic at [%d][%d]", w, r, e)
				}
			}
		}
	}
}

func TestRoundConstantsDistinctAcrossWidths(t *testing.T) {
	p := bn254()
	// The first constant of t=2 and t=3 should differ (different base offsets)
	rc2 := NewRoundConstants(2, p)
	rc3 := NewRoundConstants(3, p)
	if rc2.Constants[0][0].Cmp(rc3.Constants[0][0]) == 0 {
		t.Error("t=2 and t=3 share first round constant — base offset not working")
	}
}

// ── Pow5 S-box tests ──────────────────────────────────────────────────────────

func TestPow5KnownValues(t *testing.T) {
	p := bn254()
	cases := []struct{ in, out int64 }{
		{0, 0},
		{1, 1},
		{2, 32},
		{3, 243},
		{4, 1024},
	}
	for _, c := range cases {
		result := Pow5(big.NewInt(c.in), p)
		expected := big.NewInt(c.out)
		if result.Cmp(expected) != 0 {
			t.Errorf("Pow5(%d): expected %d, got %s", c.in, c.out, result)
		}
	}
}

func TestPow5InField(t *testing.T) {
	p := bn254()
	x := new(big.Int).Sub(p, big.NewInt(1))
	result := Pow5(x, p)
	mustInField(t, "Pow5(p-1)", result, p)
}

func TestPow5Goldilocks(t *testing.T) {
	p := new(big.Int).Set(GoldilocksField)
	result := Pow5(big.NewInt(2), p)
	expected := big.NewInt(32)
	if result.Cmp(expected) != 0 {
		t.Errorf("Pow5(2) Goldilocks: expected 32, got %s", result)
	}
	mustInField(t, "Pow5 Goldilocks", result, p)
}

// ── ApplyPermutation tests ────────────────────────────────────────────────────

func TestApplyPermutationDeterminism(t *testing.T) {
	p := bn254()
	for _, w := range supportedWidths {
		rc := NewRoundConstants(w, p)
		mds := NewCirculantMDS(w, p)

		s1 := makeState(w, p)
		s2 := makeState(w, p)

		ApplyPermutation(s1, rc, mds)
		ApplyPermutation(s2, rc, mds)

		for i := range s1 {
			if s1[i].Cmp(s2[i]) != 0 {
				t.Errorf("t=%d: ApplyPermutation not deterministic at [%d]", w, i)
			}
		}
	}
}

func TestApplyPermutationOutputInField(t *testing.T) {
	p := bn254()
	for _, w := range supportedWidths {
		rc := NewRoundConstants(w, p)
		mds := NewCirculantMDS(w, p)
		state := makeState(w, p)
		ApplyPermutation(state, rc, mds)
		for i, v := range state {
			mustInField(t, fmt.Sprintf("perm output t=%d [%d]", w, i), v, p)
		}
	}
}

func TestApplyPermutationDifferentInputsDifferentOutput(t *testing.T) {
	p := bn254()
	for _, w := range supportedWidths {
		rc := NewCirculantMDS(w, p) // reuse just to get width
		_ = rc
		rcC := NewRoundConstants(w, p)
		mds := NewCirculantMDS(w, p)

		s1 := makeState(w, p)
		s2 := makeStateOffset(w, p, 50)

		ApplyPermutation(s1, rcC, mds)
		ApplyPermutation(s2, rcC, mds)

		allSame := true
		for i := range s1 {
			if s1[i].Cmp(s2[i]) != 0 {
				allSame = false
				break
			}
		}
		if allSame {
			t.Errorf("t=%d: different inputs produced same permutation output", w)
		}
	}
}

func TestApplyPermutationAvalanche(t *testing.T) {
	p := bn254()
	changed, total := 0, 0

	for _, w := range supportedWidths {
		rc := NewRoundConstants(w, p)
		mds := NewCirculantMDS(w, p)

		for i := 0; i < 50; i++ {
			base := KConstantField(i%512, p)
			flipped := new(big.Int).Xor(base, big.NewInt(1))
			flipped.Mod(flipped, p)

			s1 := makeState(w, p)
			s1[0] = base
			s2 := makeState(w, p)
			s2[0] = flipped

			ApplyPermutation(s1, rc, mds)
			ApplyPermutation(s2, rc, mds)

			for j := range s1 {
				diff := new(big.Int).Xor(s1[j], s2[j])
				changed += bits.OnesCount64(diff.Uint64())
				total += 64
			}
		}
	}
	pct := float64(changed) / float64(total) * 100.0
	t.Logf("ApplyPermutation avalanche across all widths: %.2f%% (%d/%d bits)", pct, changed, total)
	if pct < 35.0 {
		t.Errorf("poor avalanche: %.2f%%", pct)
	}
}

// ── PermuteState tests ────────────────────────────────────────────────────────

func TestPermuteStateDeterminism(t *testing.T) {
	p := bn254()
	state := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}

	out1 := PermuteState(state, ModeCirculant, p)
	out2 := PermuteState(state, ModeCirculant, p)

	for i := range out1 {
		if out1[i].Cmp(out2[i]) != 0 {
			t.Errorf("PermuteState not deterministic at [%d]", i)
		}
	}
}

func TestPermuteStateDoesNotMutateInput(t *testing.T) {
	p := bn254()
	orig := []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30)}
	input := []*big.Int{
		new(big.Int).Set(orig[0]),
		new(big.Int).Set(orig[1]),
		new(big.Int).Set(orig[2]),
	}
	PermuteState(input, ModeCirculant, p)
	for i := range input {
		if input[i].Cmp(orig[i]) != 0 {
			t.Errorf("PermuteState mutated input[%d]", i)
		}
	}
}

func TestPermuteStateModesProduceDifferentOutput(t *testing.T) {
	p := bn254()
	state := []*big.Int{big.NewInt(42), big.NewInt(43), big.NewInt(44)}
	outC := PermuteState(state, ModeCirculant, p)
	outD := PermuteState(state, ModeCiDerived, p)

	allSame := true
	for i := range outC {
		if outC[i].Cmp(outD[i]) != 0 {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("circulant and ci-derived PermuteState produced identical output")
	}
}

// ── Goldilocks field tests ────────────────────────────────────────────────────

func TestGoldilocksFieldValue(t *testing.T) {
	// p = 2^64 - 2^32 + 1
	p := new(big.Int).Set(GoldilocksField)
	two64 := new(big.Int).Lsh(big.NewInt(1), 64)
	two32 := new(big.Int).Lsh(big.NewInt(1), 32)
	expected := new(big.Int).Sub(two64, two32)
	expected.Add(expected, big.NewInt(1))
	if p.Cmp(expected) != 0 {
		t.Errorf("GoldilocksField value incorrect: got %s, expected %s", p, expected)
	}
}

func TestGoldilocksKConstants(t *testing.T) {
	p := new(big.Int).Set(GoldilocksField)
	constants := GenerateKConstants(64, p)
	for i, c := range constants {
		mustInField(t, fmt.Sprintf("Goldilocks K[%d]", i), c, p)
		mustPositive(t, fmt.Sprintf("Goldilocks K[%d]", i), c)
	}
}

func TestGoldilocksCirculantMDS(t *testing.T) {
	p := new(big.Int).Set(GoldilocksField)
	for _, w := range supportedWidths {
		m := NewCirculantMDS(w, p)
		if !m.IsMDS() {
			t.Errorf("Goldilocks circulant t=%d: failed MDS property", w)
		}
	}
}

func TestGoldiclocksRoundConstants(t *testing.T) {
	p := new(big.Int).Set(GoldilocksField)
	for _, w := range supportedWidths {
		rc := NewRoundConstants(w, p)
		for r, row := range rc.Constants {
			for e, v := range row {
				mustInField(t, fmt.Sprintf("Goldilocks RC t=%d [%d][%d]", w, r, e), v, p)
			}
		}
	}
}

func TestGoldilocksPermutation(t *testing.T) {
	p := new(big.Int).Set(GoldilocksField)
	rc := NewRoundConstants(3, p)
	mds := NewCirculantMDS(3, p)
	state := makeState(3, p)
	ApplyPermutation(state, rc, mds)
	for i, v := range state {
		mustInField(t, fmt.Sprintf("Goldilocks perm output[%d]", i), v, p)
	}
}

func TestNewSpongeGoldilocks(t *testing.T) {
	s := NewSpongeGoldilocks(ModeCirculant)
	if s.CurrentWidth != 2 {
		t.Errorf("Goldilocks sponge: expected initial width 2, got %d", s.CurrentWidth)
	}
	p := new(big.Int).Set(GoldilocksField)
	for i, v := range s.State {
		mustInField(t, fmt.Sprintf("Goldilocks initial state[%d]", i), v, p)
	}
}

func TestGoldilocksSpongeHash(t *testing.T) {
	p := new(big.Int).Set(GoldilocksField)
	s := NewSpongeGoldilocks(ModeCirculant)
	inputs := []*big.Int{
		new(big.Int).Mod(big.NewInt(12345), p),
		new(big.Int).Mod(big.NewInt(67890), p),
	}
	out := s.Hash(inputs, 4)
	if len(out) != 4 {
		t.Errorf("Goldilocks hash: expected 4 outputs, got %d", len(out))
	}
	for i, v := range out {
		mustInField(t, fmt.Sprintf("Goldilocks hash out[%d]", i), v, p)
	}
}

func TestGoldilocksSpongeAvalanche(t *testing.T) {
	p := new(big.Int).Set(GoldilocksField)
	changed, total := 0, 0

	for i := 0; i < 100; i++ {
		base := new(big.Int).Mod(KConstantField(i%512, bn254()), p)
		flipped := new(big.Int).Xor(base, big.NewInt(1))
		flipped.Mod(flipped, p)

		s1 := NewSpongeGoldilocks(ModeCirculant)
		s2 := NewSpongeGoldilocks(ModeCirculant)

		out1 := s1.Hash([]*big.Int{base}, 4)
		out2 := s2.Hash([]*big.Int{flipped}, 4)

		for j := range out1 {
			diff := new(big.Int).Xor(out1[j], out2[j])
			changed += bits.OnesCount64(diff.Uint64())
			total += 64
		}
	}
	pct := float64(changed) / float64(total) * 100.0
	t.Logf("Goldilocks avalanche: %.2f%% (%d/%d bits)", pct, changed, total)
	if pct < 35.0 {
		t.Errorf("Goldilocks: poor avalanche %.2f%%", pct)
	}
}

// ── Parameter summary test ────────────────────────────────────────────────────

func TestParameterSummaryNonEmpty(t *testing.T) {
	s := ParameterSummary(bn254())
	if len(s) == 0 {
		t.Error("ParameterSummary returned empty string")
	}
	t.Log("\n" + s)
}
