// permutation.go — Standalone HADES/Poseidon2-style permutation engine
//
// This is the canonical permutation implementation for ci-poseidon.
// sponge.go delegates to ApplyPermutation() rather than duplicating logic.
//
// Round structure (HADES design):
//   1. Initial AddRoundConstants
//   2. rf/2 full rounds  (S-box on ALL elements + MDS)
//   3. rp partial rounds (S-box on FIRST element only + MDS)
//   4. rf/2 full rounds  (S-box on ALL elements + MDS)
//
// S-box: x^5 mod p — standard for BN254, BLS12-381, Goldilocks
// MDS:   provided by mds.go — circulant baseline or ci-derived
//
// Round constants are derived from the K-constant generator (Ci = 85/27)
// ensuring the same rational foundation throughout.
//
// Author:  Christopher Seekins — Harmony Worldwide / HealChain
// Package: github.com/karmaxul/ci-poseidon
// Date:    June 2026

package ciposeidon

import (
	"fmt"
	"math/big"
)

// ── Round parameters ──────────────────────────────────────────────────────────

// PermutationParams holds tuned security parameters per state width.
// Values are calibrated for ~128-bit security in ZK settings,
// based on Poseidon2 literature and adapted for the variable-width design.
//
// Wider states need fewer partial rounds because the MDS matrix provides
// stronger inter-element mixing at each step.
type PermutationParams struct {
	Width         int
	FullRounds    int // rf — full S-box applied to all elements
	PartialRounds int // rp — S-box applied to first element only
}

// GetPermutationParams returns the tuned round parameters for a given width.
func GetPermutationParams(width int) PermutationParams {
	switch width {
	case 2:
		return PermutationParams{Width: 2, FullRounds: 8, PartialRounds: 56}
	case 3:
		return PermutationParams{Width: 3, FullRounds: 8, PartialRounds: 40}
	case 4:
		return PermutationParams{Width: 4, FullRounds: 8, PartialRounds: 32}
	case 6:
		return PermutationParams{Width: 6, FullRounds: 8, PartialRounds: 24}
	default:
		return PermutationParams{Width: width, FullRounds: 8, PartialRounds: 32}
	}
}

// TotalRounds returns the total number of rounds (rf + rp).
func (pp PermutationParams) TotalRounds() int {
	return pp.FullRounds + pp.PartialRounds
}

// ConstantsNeeded returns the total number of K-constants required.
// Each round needs `width` constants, plus one initial AddRoundConstants.
func (pp PermutationParams) ConstantsNeeded() int {
	return (pp.TotalRounds() + 1) * pp.Width
}

// ── Round constants ───────────────────────────────────────────────────────────

// RoundConstants holds precomputed constants organised per round.
// rc.Constants[round][element] is the constant for element `element`
// in round `round`.
type RoundConstants struct {
	Params    PermutationParams
	Constants [][]*big.Int // [round][element]
}

// NewRoundConstants generates round constants for the given width and field.
// All constants are derived from the K-constant generator (Ci = 85/27),
// offset by a base index so different widths use different constant slices.
func NewRoundConstants(width int, fieldPrime *big.Int) *RoundConstants {
	params := GetPermutationParams(width)
	totalRounds := params.TotalRounds() + 1 // +1 for initial AddRoundConstants

	// Base offset per width ensures distinct constants across widths
	// t=2→0, t=3→128, t=4→256, t=6→384
	baseOffset := map[int]int{2: 0, 3: 128, 4: 256, 6: 384}
	offset, ok := baseOffset[width]
	if !ok {
		offset = 0
	}

	constants := make([][]*big.Int, totalRounds)
	kIdx := offset
	for r := 0; r < totalRounds; r++ {
		constants[r] = make([]*big.Int, width)
		for e := 0; e < width; e++ {
			constants[r][e] = KConstantField(kIdx%512, fieldPrime)
			kIdx++
		}
	}

	return &RoundConstants{
		Params:    params,
		Constants: constants,
	}
}

// ── S-box ─────────────────────────────────────────────────────────────────────

// Pow5 computes x^5 mod p — the standard Poseidon2 S-box.
// x^5 is used because gcd(5, p-1) = 1 for BN254, BLS12-381, and Goldilocks,
// guaranteeing the S-box is a bijection (invertible) over the field.
func Pow5(x, p *big.Int) *big.Int {
	x2 := new(big.Int).Mul(x, x)
	x2.Mod(x2, p)
	x4 := new(big.Int).Mul(x2, x2)
	x4.Mod(x4, p)
	x5 := new(big.Int).Mul(x4, x)
	x5.Mod(x5, p)
	return x5
}

// ── Core permutation ──────────────────────────────────────────────────────────

// ApplyPermutation performs one full HADES/Poseidon2-style permutation
// on the state vector in-place.
//
// The state slice is modified directly. The caller must not reuse
// the original slice values after calling this function.
//
// Round structure:
//   round 0:        initial AddRoundConstants (no S-box, no MDS)
//   rounds 1..rf/2: full rounds
//   rounds rf/2+1..rf/2+rp: partial rounds
//   rounds rf/2+rp+1..rf+rp: full rounds
func ApplyPermutation(state []*big.Int, rc *RoundConstants, mds *MDSMatrix) {
	if len(state) != rc.Params.Width {
		panic(fmt.Sprintf("permutation: state width %d != params width %d",
			len(state), rc.Params.Width))
	}
	if mds.Width != rc.Params.Width {
		panic(fmt.Sprintf("permutation: MDS width %d != params width %d",
			mds.Width, rc.Params.Width))
	}

	p := mds.FieldPrime
	rf := rc.Params.FullRounds
	rp := rc.Params.PartialRounds
	roundIdx := 0

	// Step 1: Initial AddRoundConstants (no S-box, no MDS)
	addRoundConstants(state, rc.Constants[roundIdx], p)
	roundIdx++

	// Step 2: First half of full rounds
	for i := 0; i < rf/2; i++ {
		fullRound(state, rc.Constants[roundIdx], mds, p)
		roundIdx++
	}

	// Step 3: Partial rounds
	for i := 0; i < rp; i++ {
		partialRound(state, rc.Constants[roundIdx], mds, p)
		roundIdx++
	}

	// Step 4: Second half of full rounds
	for i := 0; i < rf/2; i++ {
		fullRound(state, rc.Constants[roundIdx], mds, p)
		roundIdx++
	}
}

// addRoundConstants adds constants to each state element mod p.
func addRoundConstants(state, constants []*big.Int, p *big.Int) {
	for i := range state {
		state[i] = new(big.Int).Add(state[i], constants[i])
		state[i].Mod(state[i], p)
	}
}

// fullRound applies: S-box to ALL elements → AddRoundConstants → MDS
func fullRound(state, constants []*big.Int, mds *MDSMatrix, p *big.Int) {
	// S-box on all elements
	for i := range state {
		state[i] = Pow5(state[i], p)
	}
	// Add round constants
	addRoundConstants(state, constants, p)
	// MDS matrix — Apply returns a new slice, copy back into state
	result := mds.Apply(state)
	copy(state, result)
}

// partialRound applies: S-box on FIRST element only → AddRoundConstants → MDS
func partialRound(state, constants []*big.Int, mds *MDSMatrix, p *big.Int) {
	// S-box on first element only
	state[0] = Pow5(state[0], p)
	// Add round constants
	addRoundConstants(state, constants, p)
	// MDS matrix
	result := mds.Apply(state)
	copy(state, result)
}

// ── Convenience: permute a fresh state ───────────────────────────────────────

// PermuteState is a convenience wrapper that generates round constants and
// an MDS matrix, then applies the full permutation. Useful for one-off hashing.
func PermuteState(state []*big.Int, mode SpongeMode, fieldPrime *big.Int) []*big.Int {
	width := len(state)
	rc := NewRoundConstants(width, fieldPrime)
	var mds *MDSMatrix
	if mode == ModeCiDerived {
		mds = NewCiDerivedMDS(width, fieldPrime)
	} else {
		mds = NewCirculantMDS(width, fieldPrime)
	}
	out := make([]*big.Int, width)
	copy(out, state)
	ApplyPermutation(out, rc, mds)
	return out
}

// ── Parameter summary ─────────────────────────────────────────────────────────

// ParameterSummary prints the permutation parameters for all supported widths.
func ParameterSummary(fieldPrime *big.Int) string {
	out := "ci-Poseidon Parameter Summary\n"
	out += fmt.Sprintf("%-8s %-6s %-8s %-8s %-12s\n",
		"Width", "rf", "rp", "Total", "Constants")
	out += "----------------------------------------\n"
	for _, w := range widthLadder {
		pp := GetPermutationParams(w)
		out += fmt.Sprintf("t=%-6d %-6d %-8d %-8d %-12d\n",
			w, pp.FullRounds, pp.PartialRounds,
			pp.TotalRounds(), pp.ConstantsNeeded())
	}
	return out
}
