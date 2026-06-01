// ci_poseidon_test.go — Tests for ci-Poseidon field constant generation
//
// Tests verify:
//   1. Determinism       — same constants every run, every platform
//   2. Field membership  — all constants are valid field elements (< p)
//   3. Non-triviality    — constants are not zero or one
//   4. Distinctness      — no two consecutive constants are equal
//   5. Cross-field       — BN254 and BLS12-381 produce different constants
//                          (as expected — same formula, different modulus)
//   6. SBox correctness  — x^5 spot checks against known values
//   7. Permutation shape — output width matches input, elements are in-field
//   8. R-constant packing — field reduction preserves non-triviality

package ciposeidon

import (
	"fmt"
	"math/big"
	"testing"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

func bn254() *big.Int { return new(big.Int).Set(BN254ScalarField) }
func bls()   *big.Int { return new(big.Int).Set(BLS12381ScalarField) }

func mustPositive(t *testing.T, label string, v *big.Int) {
	t.Helper()
	if v.Sign() <= 0 {
		t.Errorf("%s: expected positive, got %s", label, v)
	}
}

func mustInField(t *testing.T, label string, v, p *big.Int) {
	t.Helper()
	if v.Cmp(p) >= 0 {
		t.Errorf("%s: value >= field prime", label)
	}
	if v.Sign() < 0 {
		t.Errorf("%s: value is negative", label)
	}
}

// ── K-constant tests ──────────────────────────────────────────────────────────

func TestKConstantDeterminism(t *testing.T) {
	p := bn254()
	a := KConstantField(0, p)
	b := KConstantField(0, p)
	if a.Cmp(b) != 0 {
		t.Errorf("K[0] not deterministic: %s != %s", a, b)
	}

	// Run again for index 42
	c := KConstantField(42, p)
	d := KConstantField(42, p)
	if c.Cmp(d) != 0 {
		t.Errorf("K[42] not deterministic")
	}
}

func TestKConstantsFieldMembership(t *testing.T) {
	p := bn254()
	constants := GenerateKConstants(512, p)
	for i, c := range constants {
		mustInField(t, fmt.Sprintf("K[%d]", i), c, p)
		mustPositive(t, fmt.Sprintf("K[%d]", i), c)
	}
}

func TestKConstantsNonTrivial(t *testing.T) {
	p := bn254()
	zero := big.NewInt(0)
	one := big.NewInt(1)
	constants := GenerateKConstants(64, p)
	for i, c := range constants {
		if c.Cmp(zero) == 0 {
			t.Errorf("K[%d] is zero — trivial constant", i)
		}
		if c.Cmp(one) == 0 {
			t.Errorf("K[%d] is one — trivial constant", i)
		}
	}
}

func TestKConstantsDistinct(t *testing.T) {
	p := bn254()
	constants := GenerateKConstants(64, p)
	for i := 0; i < len(constants)-1; i++ {
		if constants[i].Cmp(constants[i+1]) == 0 {
			t.Errorf("K[%d] == K[%d] — consecutive constants should differ", i, i+1)
		}
	}
}

func TestKConstantsCrossField(t *testing.T) {
	// Same index must produce different values in different fields
	k0_bn254 := KConstantField(0, bn254())
	k0_bls   := KConstantField(0, bls())
	if k0_bn254.Cmp(k0_bls) == 0 {
		t.Error("K[0] identical across BN254 and BLS12-381 — unexpected")
	}

	// Both must be valid in their respective fields
	mustInField(t, "K[0] BN254",    k0_bn254, bn254())
	mustInField(t, "K[0] BLS12381", k0_bls,   bls())
}

func TestKConstantsBLS12381FieldMembership(t *testing.T) {
	p := bls()
	constants := GenerateKConstants(64, p)
	for i, c := range constants {
		mustInField(t, fmt.Sprintf("K[%d] BLS", i), c, p)
	}
}

// ── R-constant tests ──────────────────────────────────────────────────────────

func TestRConstantFieldMembership(t *testing.T) {
	p := bn254()
	// Use Ci element (element 120) values from the resonance matrix
	// tHz=391.5 → tHz10=3915, nm=688.5 → nm10=6885, nX=152, nY=147
	r := RConstantField(3915, 6885, 152, 147, p)
	mustInField(t, "R[Ci element]", r, p)
	mustPositive(t, "R[Ci element]", r)
}

func TestRConstantDeterminism(t *testing.T) {
	p := bn254()
	a := RConstantField(3915, 6885, 152, 147, p)
	b := RConstantField(3915, 6885, 152, 147, p)
	if a.Cmp(b) != 0 {
		t.Error("R-constant not deterministic")
	}
}

func TestRConstantDistinctInputs(t *testing.T) {
	p := bn254()
	// Two different elements should produce different R-constants
	r1 := RConstantField(3915, 6885, 152, 147, p) // Ci element
	r2 := RConstantField(5250, 4050, 140, 159, p) // different element
	if r1.Cmp(r2) == 0 {
		t.Error("different inputs produced same R-constant")
	}
}

// ── SBox tests ────────────────────────────────────────────────────────────────

func TestSBoxKnownValues(t *testing.T) {
	p := bn254()

	// 2^5 = 32
	two := big.NewInt(2)
	result := SBox(two, p)
	expected := big.NewInt(32)
	if result.Cmp(expected) != 0 {
		t.Errorf("SBox(2): expected 32, got %s", result)
	}

	// 3^5 = 243
	three := big.NewInt(3)
	result = SBox(three, p)
	expected = big.NewInt(243)
	if result.Cmp(expected) != 0 {
		t.Errorf("SBox(3): expected 243, got %s", result)
	}

	// 0^5 = 0
	zero := big.NewInt(0)
	result = SBox(zero, p)
	if result.Sign() != 0 {
		t.Errorf("SBox(0): expected 0, got %s", result)
	}

	// 1^5 = 1
	one := big.NewInt(1)
	result = SBox(one, p)
	if result.Cmp(one) != 0 {
		t.Errorf("SBox(1): expected 1, got %s", result)
	}
}

func TestSBoxResultInField(t *testing.T) {
	p := bn254()
	// Test with a large value close to p
	x := new(big.Int).Sub(p, big.NewInt(1)) // p-1
	result := SBox(x, p)
	mustInField(t, "SBox(p-1)", result, p)
}

// ── Permutation tests ─────────────────────────────────────────────────────────

func TestPermuteOutputWidth(t *testing.T) {
	p := bn254()
	params := NewPoseidon2Params(3, 8, 56, p)

	state := []*big.Int{
		big.NewInt(1),
		big.NewInt(2),
		big.NewInt(3),
	}
	out := params.Permute(state)
	if len(out) != 3 {
		t.Errorf("output width: expected 3, got %d", len(out))
	}
}

func TestPermuteOutputInField(t *testing.T) {
	p := bn254()
	params := NewPoseidon2Params(3, 8, 56, p)

	state := []*big.Int{
		big.NewInt(100),
		big.NewInt(200),
		big.NewInt(300),
	}
	out := params.Permute(state)
	for i, v := range out {
		mustInField(t, fmt.Sprintf("permute output[%d]", i), v, p)
	}
}

func TestPermuteDeterminism(t *testing.T) {
	p := bn254()
	params := NewPoseidon2Params(3, 8, 56, p)

	state1 := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	state2 := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}

	out1 := params.Permute(state1)
	out2 := params.Permute(state2)

	for i := range out1 {
		if out1[i].Cmp(out2[i]) != 0 {
			t.Errorf("permute not deterministic at index %d", i)
		}
	}
}

func TestPermuteDoesNotMutateInput(t *testing.T) {
	p := bn254()
	params := NewPoseidon2Params(3, 8, 56, p)

	orig := []*big.Int{big.NewInt(42), big.NewInt(43), big.NewInt(44)}
	input := []*big.Int{
		new(big.Int).Set(orig[0]),
		new(big.Int).Set(orig[1]),
		new(big.Int).Set(orig[2]),
	}
	params.Permute(input)

	for i := range input {
		if input[i].Cmp(orig[i]) != 0 {
			t.Errorf("Permute mutated input[%d]", i)
		}
	}
}

func TestPermuteDifferentInputsDifferentOutputs(t *testing.T) {
	p := bn254()
	params := NewPoseidon2Params(3, 8, 56, p)

	out1 := params.Permute([]*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)})
	out2 := params.Permute([]*big.Int{big.NewInt(4), big.NewInt(5), big.NewInt(6)})

	identical := true
	for i := range out1 {
		if out1[i].Cmp(out2[i]) != 0 {
			identical = false
			break
		}
	}
	if identical {
		t.Error("different inputs produced identical permutation outputs")
	}
}

// ── Parameter tests ───────────────────────────────────────────────────────────

func TestNewPoseidon2ParamsConstantCount(t *testing.T) {
	p := bn254()
	params := NewPoseidon2Params(3, 8, 56, p)
	expected := (8 + 56) * 3 // 192
	if len(params.RoundConstants) != expected {
		t.Errorf("expected %d round constants, got %d", expected, len(params.RoundConstants))
	}
}

func TestFieldElementHex(t *testing.T) {
	x := big.NewInt(255)
	h := FieldElementHex(x)
	if len(h) != 64 {
		t.Errorf("hex length: expected 64, got %d", len(h))
	}
	expected := "00000000000000000000000000000000000000000000000000000000000000ff"
	if h != expected {
		t.Errorf("hex mismatch: got %s", h)
	}
}
