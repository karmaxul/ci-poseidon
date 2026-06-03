// plonk_bench_test.go — PLONK constraint count and prover/verifier benchmarks
//
// Measures PlonK gate counts and proof generation/verification time for
// ci-poseidon circuits at each supported width (t=2,3,4,6) using gnark's
// PLONK backend over BN254.
//
// PLONK uses a different arithmetization from R1CS:
//   - R1CS counts multiplication gates (each x^5 S-box = 3 constraints)
//   - PLONK counts "gates" which are (a·b + c) style — slightly different
//     accounting but the x^5 S-box structure is identical
//
// This file is the direct parallel to gnark_bench_test.go — same circuits,
// same witnesses, same field — only the backend changes (plonk vs groth16).
//
// Key expected findings:
//   - Gate counts match R1CS constraint counts closely (same circuit structure)
//   - PLONK proof size: ~736 bytes (larger than Groth16's ~127 bytes)
//   - PLONK verify: O(log n) vs Groth16's O(1) — slightly slower verify
//   - PLONK prove: typically faster than Groth16 for small circuits
//   - Flat gate profile across t=2→t=6 (same width hypothesis test)
//
// Run gate count test:
//   go test -v -run TestPlonkConstraintCount
//
// Run proof correctness:
//   go test -v -run TestPlonkProofCorrect_t3 -timeout 120s
//
// Run benchmarks:
//   go test -bench=BenchmarkPlonk -benchmem -timeout 300s
//
// Run full summary:
//   go test -v -run TestPlonkSummary -timeout 120s
//
// Author:  Christopher Seekins — Harmony Worldwide / HealChain
// Package: github.com/karmaxul/ci-poseidon
// Date:    June 2026

package ciposeidon

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/plonk"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/scs"
	"github.com/consensys/gnark/test/unsafekzg"
)

// ── PLONK constraint count test ───────────────────────────────────────────────

// TestPlonkConstraintCount compiles each circuit under the PLONK (SCS) backend
// and reports the gate count. Compare against R1CS counts from gnark_bench_test.go:
//
//	t=2: R1CS=216  →  PLONK gate count (measured here)
//	t=3: R1CS=192  →  PLONK gate count
//	t=4: R1CS=192  →  PLONK gate count
//	t=6: R1CS=216  →  PLONK gate count
func TestPlonkConstraintCount(t *testing.T) {
	t.Log("═══════════════════════════════════════════════════════════")
	t.Log("  PLONK (SCS) GATE COUNT — ci-poseidon, BN254")
	t.Log("  Compare: R1CS counts were t2=216 t3=192 t4=192 t6=216")
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

	r1csCounts := map[int]int{2: 216, 3: 192, 4: 192, 6: 216}

	for _, c := range cases {
		cs, err := frontend.Compile(ecc.BN254.ScalarField(), scs.NewBuilder, c.circuit)
		if err != nil {
			t.Errorf("t=%d: compile error: %v", c.width, err)
			continue
		}
		plonkGates := cs.GetNbConstraints()
		r1cs := r1csCounts[c.width]
		delta := plonkGates - r1cs
		sign := "+"
		if delta < 0 {
			sign = ""
		}
		t.Logf("  t=%-2d  PLONK gates: %5d  R1CS: %5d  delta: %s%d",
			c.width, plonkGates, r1cs, sign, delta)
	}
	t.Log("═══════════════════════════════════════════════════════════")
}

// ── PLONK proof correctness ───────────────────────────────────────────────────

// TestPlonkProofCorrect_t3 runs a full PLONK prove+verify cycle for t=3.
// Uses unsafekzg for the SRS — suitable for testing only, not production.
func TestPlonkProofCorrect_t3(t *testing.T) {
	inputs, outputs := witnessForWidth(3)

	circuit := &CiPoseidonCircuit_t3{}
	cs, err := frontend.Compile(ecc.BN254.ScalarField(), scs.NewBuilder, circuit)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	srs, srsLagrange, err := unsafekzg.NewSRS(cs)
	if err != nil {
		t.Fatalf("SRS: %v", err)
	}

	pk, vk, err := plonk.Setup(cs, srs, srsLagrange)
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

	proof, err := plonk.Prove(cs, pk, w)
	if err != nil {
		t.Fatalf("prove: %v", err)
	}

	pubW, err := w.Public()
	if err != nil {
		t.Fatalf("public witness: %v", err)
	}

	err = plonk.Verify(proof, vk, pubW)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	t.Log("✓ PLONK proof correct for t=3 ci-poseidon permutation")
}

// TestPlonkProofCorrect_t6 verifies the widest supported state works correctly.
func TestPlonkProofCorrect_t6(t *testing.T) {
	inputs, outputs := witnessForWidth(6)

	circuit := &CiPoseidonCircuit_t6{}
	cs, err := frontend.Compile(ecc.BN254.ScalarField(), scs.NewBuilder, circuit)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	srs, srsLagrange, err := unsafekzg.NewSRS(cs)
	if err != nil {
		t.Fatalf("SRS: %v", err)
	}

	pk, vk, err := plonk.Setup(cs, srs, srsLagrange)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	witness := &CiPoseidonCircuit_t6{
		In:  [6]frontend.Variable{inputs[0], inputs[1], inputs[2], inputs[3], inputs[4], inputs[5]},
		Out: [6]frontend.Variable{outputs[0], outputs[1], outputs[2], outputs[3], outputs[4], outputs[5]},
	}
	w, err := frontend.NewWitness(witness, ecc.BN254.ScalarField())
	if err != nil {
		t.Fatalf("witness: %v", err)
	}

	proof, err := plonk.Prove(cs, pk, w)
	if err != nil {
		t.Fatalf("prove: %v", err)
	}

	pubW, err := w.Public()
	if err != nil {
		t.Fatalf("public witness: %v", err)
	}

	err = plonk.Verify(proof, vk, pubW)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	t.Log("✓ PLONK proof correct for t=6 ci-poseidon permutation")
}

// ── PLONK prover benchmarks ───────────────────────────────────────────────────

func benchmarkPlonk(b *testing.B, width int, circuit frontend.Circuit) {
	cs, err := frontend.Compile(ecc.BN254.ScalarField(), scs.NewBuilder, circuit)
	if err != nil {
		b.Fatalf("compile: %v", err)
	}

	srs, srsLagrange, err := unsafekzg.NewSRS(cs)
	if err != nil {
		b.Fatalf("SRS: %v", err)
	}

	pk, _, err := plonk.Setup(cs, srs, srsLagrange)
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
		_, err := plonk.Prove(cs, pk, w)
		if err != nil {
			b.Fatalf("prove: %v", err)
		}
	}
}

func benchmarkPlonkVerify(b *testing.B, width int, circuit frontend.Circuit) {
	cs, err := frontend.Compile(ecc.BN254.ScalarField(), scs.NewBuilder, circuit)
	if err != nil {
		b.Fatalf("compile: %v", err)
	}

	srs, srsLagrange, err := unsafekzg.NewSRS(cs)
	if err != nil {
		b.Fatalf("SRS: %v", err)
	}

	pk, vk, err := plonk.Setup(cs, srs, srsLagrange)
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

	proof, err := plonk.Prove(cs, pk, fullW)
	if err != nil {
		b.Fatalf("prove: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := plonk.Verify(proof, vk, pubW)
		if err != nil {
			b.Fatalf("verify: %v", err)
		}
	}
}

func BenchmarkPlonkProve_t2(b *testing.B)  { benchmarkPlonk(b, 2, &CiPoseidonCircuit_t2{}) }
func BenchmarkPlonkProve_t3(b *testing.B)  { benchmarkPlonk(b, 3, &CiPoseidonCircuit_t3{}) }
func BenchmarkPlonkProve_t4(b *testing.B)  { benchmarkPlonk(b, 4, &CiPoseidonCircuit_t4{}) }
func BenchmarkPlonkProve_t6(b *testing.B)  { benchmarkPlonk(b, 6, &CiPoseidonCircuit_t6{}) }

func BenchmarkPlonkVerify_t2(b *testing.B) { benchmarkPlonkVerify(b, 2, &CiPoseidonCircuit_t2{}) }
func BenchmarkPlonkVerify_t3(b *testing.B) { benchmarkPlonkVerify(b, 3, &CiPoseidonCircuit_t3{}) }
func BenchmarkPlonkVerify_t4(b *testing.B) { benchmarkPlonkVerify(b, 4, &CiPoseidonCircuit_t4{}) }
func BenchmarkPlonkVerify_t6(b *testing.B) { benchmarkPlonkVerify(b, 6, &CiPoseidonCircuit_t6{}) }

// ── Summary test ──────────────────────────────────────────────────────────────

// TestPlonkSummary compiles all four circuits and prints a comparison table
// of PLONK gate counts vs Groth16 R1CS counts, plus proof size.
func TestPlonkSummary(t *testing.T) {
	t.Log("")
	t.Log("╔══════════════════════════════════════════════════════════════╗")
	t.Log("║     PLONK BENCHMARK SUMMARY — ci-poseidon, BN254            ║")
	t.Log("║     June 2026                                                ║")
	t.Log("╠══════════════════════════════════════════════════════════════╣")

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

	r1csCounts := map[int]int{2: 216, 3: 192, 4: 192, 6: 216}

	t.Log("║  width  PLONK gates  R1CS  delta  rf  rp                    ║")
	t.Log("║  ─────  ──────────  ────  ─────  ──  ──                    ║")

	for _, c := range cases {
		cs, err := frontend.Compile(ecc.BN254.ScalarField(), scs.NewBuilder, c.circuit)
		if err != nil {
			t.Errorf("t=%d compile: %v", c.width, err)
			continue
		}
		gates := cs.GetNbConstraints()
		r1cs := r1csCounts[c.width]
		pp := GetPermutationParams(c.width)
		delta := gates - r1cs
		sign := "+"
		if delta < 0 {
			sign = ""
		}
		t.Logf("║  t=%-4d  %5d       %4d  %s%-4d  %2d  %2d                  ║",
			c.width, gates, r1cs, sign, delta, pp.FullRounds, pp.PartialRounds)
	}

	// Measure proof size for t=3
	circuit3 := &CiPoseidonCircuit_t3{}
	cs3, err := frontend.Compile(ecc.BN254.ScalarField(), scs.NewBuilder, circuit3)
	if err == nil {
		srs, srsLagrange, err := unsafekzg.NewSRS(cs3)
		if err == nil {
			pk, vk, err := plonk.Setup(cs3, srs, srsLagrange)
			if err == nil {
				inputs, outputs := witnessForWidth(3)
				witness := &CiPoseidonCircuit_t3{
					In:  [3]frontend.Variable{inputs[0], inputs[1], inputs[2]},
					Out: [3]frontend.Variable{outputs[0], outputs[1], outputs[2]},
				}
				w, _ := frontend.NewWitness(witness, ecc.BN254.ScalarField())
				proof, err := plonk.Prove(cs3, pk, w)
				if err == nil {
					_ = vk // suppress unused warning
					// Serialize proof to measure size
					var buf []byte
					var sizeBuf bytes.Buffer
					if _, err := proof.WriteTo(&sizeBuf); err == nil {
						buf = sizeBuf.Bytes()
					}
					t.Log("╠══════════════════════════════════════════════════════════════╣")
					t.Logf("║  PLONK proof size (t=3, BN254):  %d bytes                    ║", len(buf))
					t.Log("║  Groth16 proof size (t=3, BN254): ~127 bytes                 ║")
					t.Log("║  Note: PLONK larger proof, faster/no trusted setup           ║")
				}
			}
		}
	}

	t.Log("╠══════════════════════════════════════════════════════════════╣")
	t.Log("║  Run benchmarks:                                             ║")
	t.Log("║    go test -bench=BenchmarkPlonk -benchmem -timeout 300s    ║")
	t.Log("╚══════════════════════════════════════════════════════════════╝")

	fmt.Println("\nPlonkish arithmetization note:")
	fmt.Println("  PLONK uses sparse constraint system (SCS) — gate counts")
	fmt.Println("  differ from R1CS but the x^5 S-box structure is identical.")
	fmt.Println("  The flat gate profile across t=2→t=6 is the key result.")
}
