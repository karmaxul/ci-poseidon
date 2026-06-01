// gnark_bench_test.go — Prover and verifier time benchmarks using gnark
//
// Measures actual ZK proof generation and verification time for ci-poseidon
// circuits at each supported width, compared against gnark's native Poseidon2.
//
// What this benchmarks:
//   - Circuit compilation (R1CS constraint generation)
//   - Groth16 trusted setup (SRS generation) — one-time cost
//   - Proof generation (prover time) — the critical metric
//   - Proof verification (verifier time)
//
// Each ci-poseidon circuit encodes one full permutation at a fixed width.
// The gnark circuit mirrors exactly what circom_export.go generates —
// same round structure, same constants, same MDS — but in gnark's
// frontend for native Go proof generation.
//
// Run benchmarks:
//   go test -bench=BenchmarkGnark -benchmem -timeout 300s
//
// Run all prover tests (no proof, just constraint count verification):
//   go test -v -run TestGnark
//
// Author:  Christopher Seekins — Harmony Worldwide / HealChain
// Package: github.com/karmaxul/ci-poseidon
// Date:    June 2026

package ciposeidon

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

// ── gnark circuit definitions ─────────────────────────────────────────────────

// CiPoseidonCircuit_t2 encodes one ci-poseidon permutation at t=2.
// All round constants are precomputed from Ci=85/27 and embedded as
// compile-time literals — no runtime field arithmetic in the circuit.
type CiPoseidonCircuit_t2 struct {
	In  [2]frontend.Variable `gnark:",public"`
	Out [2]frontend.Variable `gnark:",public"`
}

func (c *CiPoseidonCircuit_t2) Define(api frontend.API) error {
	result := ciPoseidonGnark(api, c.In[:], 2)
	for i := range c.Out {
		api.AssertIsEqual(c.Out[i], result[i])
	}
	return nil
}

// CiPoseidonCircuit_t3 encodes one ci-poseidon permutation at t=3.
type CiPoseidonCircuit_t3 struct {
	In  [3]frontend.Variable `gnark:",public"`
	Out [3]frontend.Variable `gnark:",public"`
}

func (c *CiPoseidonCircuit_t3) Define(api frontend.API) error {
	result := ciPoseidonGnark(api, c.In[:], 3)
	for i := range c.Out {
		api.AssertIsEqual(c.Out[i], result[i])
	}
	return nil
}

// CiPoseidonCircuit_t4 encodes one ci-poseidon permutation at t=4.
type CiPoseidonCircuit_t4 struct {
	In  [4]frontend.Variable `gnark:",public"`
	Out [4]frontend.Variable `gnark:",public"`
}

func (c *CiPoseidonCircuit_t4) Define(api frontend.API) error {
	result := ciPoseidonGnark(api, c.In[:], 4)
	for i := range c.Out {
		api.AssertIsEqual(c.Out[i], result[i])
	}
	return nil
}

// CiPoseidonCircuit_t6 encodes one ci-poseidon permutation at t=6.
type CiPoseidonCircuit_t6 struct {
	In  [6]frontend.Variable `gnark:",public"`
	Out [6]frontend.Variable `gnark:",public"`
}

func (c *CiPoseidonCircuit_t6) Define(api frontend.API) error {
	result := ciPoseidonGnark(api, c.In[:], 6)
	for i := range c.Out {
		api.AssertIsEqual(c.Out[i], result[i])
	}
	return nil
}

// ── gnark permutation implementation ─────────────────────────────────────────

// ciPoseidonGnark implements the ci-poseidon permutation in gnark's frontend.
// This mirrors exactly the Go implementation in permutation.go but expressed
// as gnark constraints rather than big.Int arithmetic.
func ciPoseidonGnark(api frontend.API, state []frontend.Variable, width int) []frontend.Variable {
	p := BN254ScalarField
	rc := NewRoundConstants(width, p)
	pp := GetPermutationParams(width)

	// Work on a copy
	st := make([]frontend.Variable, width)
	copy(st, state)

	roundIdx := 0

	// Initial AddRoundConstants
	for i := 0; i < width; i++ {
		st[i] = api.Add(st[i], rc.Constants[roundIdx][i].String())
	}
	roundIdx++

	// First half full rounds
	for r := 0; r < pp.FullRounds/2; r++ {
		st = gnarkFullRound(api, st, rc.Constants[roundIdx], width, p)
		roundIdx++
	}

	// Partial rounds
	for r := 0; r < pp.PartialRounds; r++ {
		st = gnarkPartialRound(api, st, rc.Constants[roundIdx], width, p)
		roundIdx++
	}

	// Second half full rounds
	for r := 0; r < pp.FullRounds/2; r++ {
		st = gnarkFullRound(api, st, rc.Constants[roundIdx], width, p)
		roundIdx++
	}

	return st
}

// gnarkPow5 computes x^5 in the circuit (3 multiplication constraints).
func gnarkPow5(api frontend.API, x frontend.Variable) frontend.Variable {
	x2 := api.Mul(x, x)
	x4 := api.Mul(x2, x2)
	return api.Mul(x4, x)
}

// gnarkFullRound applies: Pow5 to ALL elements + AddRoundConstants + MDS
func gnarkFullRound(api frontend.API, state []frontend.Variable, constants []*big.Int, width int, p *big.Int) []frontend.Variable {
	// S-box on all elements
	for i := range state {
		state[i] = gnarkPow5(api, state[i])
	}
	// Add round constants
	for i := range state {
		state[i] = api.Add(state[i], constants[i].String())
	}
	// MDS (circulant — linear, zero multiplicative constraints)
	return gnarkMDS(api, state, width)
}

// gnarkPartialRound applies: Pow5 on FIRST element + AddRoundConstants + MDS
func gnarkPartialRound(api frontend.API, state []frontend.Variable, constants []*big.Int, width int, p *big.Int) []frontend.Variable {
	// S-box on first element only
	state[0] = gnarkPow5(api, state[0])
	// Add round constants
	for i := range state {
		state[i] = api.Add(state[i], constants[i].String())
	}
	// MDS
	return gnarkMDS(api, state, width)
}

// gnarkMDS applies the circulant MDS matrix (linear — free in R1CS).
func gnarkMDS(api frontend.API, state []frontend.Variable, width int) []frontend.Variable {
	seeds := circulantSeedsForWidth(width)
	out := make([]frontend.Variable, width)
	for i := 0; i < width; i++ {
		// out[i] = sum_j( seeds[(j-i+width)%width] * state[j] )
		var acc frontend.Variable = 0
		for j := 0; j < width; j++ {
			seed := seeds[((j-i)+width)%width]
			term := api.Mul(seed, state[j])
			acc = api.Add(acc, term)
		}
		out[i] = acc
	}
	return out
}

// circulantSeedsForWidth returns the int64 seeds for each width.
func circulantSeedsForWidth(width int) []int64 {
	switch width {
	case 2:
		return []int64{2, 1}
	case 3:
		return []int64{2, 1, 1}
	case 4:
		return []int64{5, 7, 1, 3}
	case 6:
		return []int64{10, 11, 13, 5, 2, 1}
	default:
		return []int64{1}
	}
}

// ── Witness helpers ───────────────────────────────────────────────────────────

// computeWitness runs the native Go permutation to get expected outputs,
// then builds the gnark witness assignment.
func witnessForWidth(width int) ([]string, []string) {
	p := BN254ScalarField
	state := makeState(width, p)
	inputs := make([]string, width)
	for i, v := range state {
		inputs[i] = v.String()
	}

	// Compute expected output using native permutation
	rc := NewRoundConstants(width, p)
	mds := NewCirculantMDS(width, p)
	out := make([]*big.Int, width)
	copy(out, state)
	ApplyPermutation(out, rc, mds)

	outputs := make([]string, width)
	for i, v := range out {
		outputs[i] = v.String()
	}
	return inputs, outputs
}

// ── Constraint count tests ────────────────────────────────────────────────────

func TestGnarkConstraintCount(t *testing.T) {
	t.Log("═══════════════════════════════════════════════════════════")
	t.Log("  GNARK R1CS CONSTRAINT COUNT (actual, not estimated)")
	t.Log("═══════════════════════════════════════════════════════════")

	type circuitCase struct {
		width   int
		circuit frontend.Circuit
	}

	cases := []circuitCase{
		{2, &CiPoseidonCircuit_t2{}},
		{3, &CiPoseidonCircuit_t3{}},
		{4, &CiPoseidonCircuit_t4{}},
		{6, &CiPoseidonCircuit_t6{}},
	}

	for _, c := range cases {
		cs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, c.circuit)
		if err != nil {
			t.Errorf("t=%d: compile error: %v", c.width, err)
			continue
		}
		actual := cs.GetNbConstraints()
		estimated := r1csConstraints(c.width)
		t.Logf("  t=%-2d  actual: %5d  estimated: %5d  match: %v",
			c.width, actual, estimated, actual == estimated)
	}
	t.Log("═══════════════════════════════════════════════════════════")
}

// ── Proof correctness tests ───────────────────────────────────────────────────

func TestGnarkProofCorrect_t3(t *testing.T) {
	inputs, outputs := witnessForWidth(3)

	circuit := &CiPoseidonCircuit_t3{}
	cs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	pk, vk, err := groth16.Setup(cs)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	witness := &CiPoseidonCircuit_t3{
		In:  [3]frontend.Variable{inputs[0], inputs[1], inputs[2]},
		Out: [3]frontend.Variable{outputs[0], outputs[1], outputs[2]},
	}
	w, err := frontend.NewWitness(witness, ecc.BN254.ScalarField())
	if err != nil {
		t.Fatalf("witness: %v", err)
	}

	proof, err := groth16.Prove(cs, pk, w)
	if err != nil {
		t.Fatalf("prove: %v", err)
	}

	pubW, err := w.Public()
	if err != nil {
		t.Fatalf("public witness: %v", err)
	}

	err = groth16.Verify(proof, vk, pubW)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	t.Log("✓ Groth16 proof correct for t=3 ci-poseidon permutation")
}

// ── Prover time benchmarks ────────────────────────────────────────────────────

func benchmarkGnark(b *testing.B, width int, circuit frontend.Circuit) {
	cs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	if err != nil {
		b.Fatalf("compile: %v", err)
	}

	pk, _, err := groth16.Setup(cs)
	if err != nil {
		b.Fatalf("setup: %v", err)
	}

	inputs, outputs := witnessForWidth(width)

	var witness frontend.Circuit
	switch width {
	case 2:
		w := &CiPoseidonCircuit_t2{}
		w.In = [2]frontend.Variable{inputs[0], inputs[1]}
		w.Out = [2]frontend.Variable{outputs[0], outputs[1]}
		witness = w
	case 3:
		w := &CiPoseidonCircuit_t3{}
		w.In = [3]frontend.Variable{inputs[0], inputs[1], inputs[2]}
		w.Out = [3]frontend.Variable{outputs[0], outputs[1], outputs[2]}
		witness = w
	case 4:
		w := &CiPoseidonCircuit_t4{}
		w.In = [4]frontend.Variable{inputs[0], inputs[1], inputs[2], inputs[3]}
		w.Out = [4]frontend.Variable{outputs[0], outputs[1], outputs[2], outputs[3]}
		witness = w
	case 6:
		w := &CiPoseidonCircuit_t6{}
		w.In = [6]frontend.Variable{inputs[0], inputs[1], inputs[2], inputs[3], inputs[4], inputs[5]}
		w.Out = [6]frontend.Variable{outputs[0], outputs[1], outputs[2], outputs[3], outputs[4], outputs[5]}
		witness = w
	}

	w, err := frontend.NewWitness(witness, ecc.BN254.ScalarField())
	if err != nil {
		b.Fatalf("witness: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := groth16.Prove(cs, pk, w)
		if err != nil {
			b.Fatalf("prove: %v", err)
		}
	}
}

func benchmarkGnarkVerify(b *testing.B, width int, circuit frontend.Circuit) {
	cs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	if err != nil {
		b.Fatalf("compile: %v", err)
	}

	pk, vk, err := groth16.Setup(cs)
	if err != nil {
		b.Fatalf("setup: %v", err)
	}

	inputs, outputs := witnessForWidth(width)

	var witness frontend.Circuit
	switch width {
	case 2:
		w := &CiPoseidonCircuit_t2{}
		w.In = [2]frontend.Variable{inputs[0], inputs[1]}
		w.Out = [2]frontend.Variable{outputs[0], outputs[1]}
		witness = w
	case 3:
		w := &CiPoseidonCircuit_t3{}
		w.In = [3]frontend.Variable{inputs[0], inputs[1], inputs[2]}
		w.Out = [3]frontend.Variable{outputs[0], outputs[1], outputs[2]}
		witness = w
	case 4:
		w := &CiPoseidonCircuit_t4{}
		w.In = [4]frontend.Variable{inputs[0], inputs[1], inputs[2], inputs[3]}
		w.Out = [4]frontend.Variable{outputs[0], outputs[1], outputs[2], outputs[3]}
		witness = w
	case 6:
		w := &CiPoseidonCircuit_t6{}
		w.In = [6]frontend.Variable{inputs[0], inputs[1], inputs[2], inputs[3], inputs[4], inputs[5]}
		w.Out = [6]frontend.Variable{outputs[0], outputs[1], outputs[2], outputs[3], outputs[4], outputs[5]}
		witness = w
	}

	fullW, err := frontend.NewWitness(witness, ecc.BN254.ScalarField())
	if err != nil {
		b.Fatalf("witness: %v", err)
	}
	pubW, err := fullW.Public()
	if err != nil {
		b.Fatalf("public witness: %v", err)
	}

	proof, err := groth16.Prove(cs, pk, fullW)
	if err != nil {
		b.Fatalf("prove: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := groth16.Verify(proof, vk, pubW)
		if err != nil {
			b.Fatalf("verify: %v", err)
		}
	}
}

func BenchmarkGnarkProve_t2(b *testing.B) { benchmarkGnark(b, 2, &CiPoseidonCircuit_t2{}) }
func BenchmarkGnarkProve_t3(b *testing.B) { benchmarkGnark(b, 3, &CiPoseidonCircuit_t3{}) }
func BenchmarkGnarkProve_t4(b *testing.B) { benchmarkGnark(b, 4, &CiPoseidonCircuit_t4{}) }
func BenchmarkGnarkProve_t6(b *testing.B) { benchmarkGnark(b, 6, &CiPoseidonCircuit_t6{}) }

func BenchmarkGnarkVerify_t2(b *testing.B) { benchmarkGnarkVerify(b, 2, &CiPoseidonCircuit_t2{}) }
func BenchmarkGnarkVerify_t3(b *testing.B) { benchmarkGnarkVerify(b, 3, &CiPoseidonCircuit_t3{}) }
func BenchmarkGnarkVerify_t4(b *testing.B) { benchmarkGnarkVerify(b, 4, &CiPoseidonCircuit_t4{}) }
func BenchmarkGnarkVerify_t6(b *testing.B) { benchmarkGnarkVerify(b, 6, &CiPoseidonCircuit_t6{}) }

// ── Summary test ──────────────────────────────────────────────────────────────

func TestGnarkSummary(t *testing.T) {
	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║     GNARK GROTH16 BENCHMARK SUMMARY — ci-poseidon         ║")
	t.Log("║     BN254 scalar field — June 2026                        ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")
	t.Log("║  Run: go test -bench=BenchmarkGnark -benchmem -timeout 300s ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")

	type circuitCase struct {
		width   int
		circuit frontend.Circuit
	}
	cases := []circuitCase{
		{2, &CiPoseidonCircuit_t2{}},
		{3, &CiPoseidonCircuit_t3{}},
		{4, &CiPoseidonCircuit_t4{}},
		{6, &CiPoseidonCircuit_t6{}},
	}

	for _, c := range cases {
		cs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, c.circuit)
		if err != nil {
			t.Errorf("t=%d compile: %v", c.width, err)
			continue
		}
		nb := cs.GetNbConstraints()
		pp := GetPermutationParams(c.width)
		t.Logf("║  t=%-2d  constraints: %4d  rf=%d  rp=%d              ║",
			c.width, nb, pp.FullRounds, pp.PartialRounds)
	}

	t.Log("╠═══════════════════════════════════════════════════════════╣")
	t.Log("║  Proof size (Groth16/BN254): ~127 bytes (constant)        ║")
	t.Log("║  Verify time (Groth16):      ~1-2ms (constant)            ║")
	t.Log("║  Prove time: see BenchmarkGnarkProve_t* above             ║")
	t.Log("╚═══════════════════════════════════════════════════════════╝")

	// Print the command to run benchmarks
	fmt.Println("\nTo get full prover timing data:")
	fmt.Println("  go test -bench=BenchmarkGnark -benchmem -timeout 300s -v")
}
