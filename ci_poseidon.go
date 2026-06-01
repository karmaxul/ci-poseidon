// ci_poseidon.go — Field-native constant generation for ci-Poseidon
//
// This package implements the ZK-friendly variant of the ci-sha4096 constant
// framework. Instead of truncating K-constants to uint64, they are reduced
// modulo a target field prime — making them native field elements suitable
// for arithmetization-oriented hash constructions (Poseidon2, HADES-style).
//
// Two target fields are supported out of the box:
//   - BN254 scalar field  (most common SNARK field — gnark, circom)
//   - BLS12-381 scalar field (used in Ethereum PoS, Zcash)
//
// The constant generation is deterministic, platform-independent, and
// verifiable from first principles using only the ratio 85/27 and the
// prime number sequence.
//
// Author:  Christopher Seekins — Harmony Worldwide / HealChain
// Package: github.com/karmaxul/ci-poseidon
// Date:    June 2026

package ciposeidon

import (
	"fmt"
	"math/big"
)

// ── Field primes ──────────────────────────────────────────────────────────────

// BN254ScalarField is the scalar field prime for the BN254 (alt-bn128) curve.
// Used by gnark, circom, Ethereum precompiles.
// p = 21888242871839275222246405745257275088548364400416034343698204186575808495617
var BN254ScalarField, _ = new(big.Int).SetString(
	"21888242871839275222246405745257275088548364400416034343698204186575808495617", 10)

// BLS12381ScalarField is the scalar field prime for the BLS12-381 curve.
// Used by Ethereum PoS, Zcash Sapling/Orchard, Filecoin.
// p = 52435875175126190479447740508185965837690552500527637822603658699938581184513
var BLS12381ScalarField, _ = new(big.Int).SetString(
	"52435875175126190479447740508185965837690552500527637822603658699938581184513", 10)

// GoldilocksField is p = 2^64 - 2^32 + 1 = 18446744069414584321
//
// Goldilocks is the standard field for STARK-based systems (Plonky2, Plonky3,
// Polygon zkEVM). Its 64-bit structure makes arithmetic extremely fast on
// modern hardware — no multi-precision arithmetic needed for the field itself.
//
// For ci-poseidon, Goldilocks support opens the FRI/STARK research direction:
// the 18-bit binary period of Ci = 85/27 may align naturally with Goldilocks
// evaluation domains (which are also powers-of-two friendly).
//
// Note: the x^5 S-box is valid over Goldilocks because gcd(5, p-1) = 1.
var GoldilocksField, _ = new(big.Int).SetString("18446744069414584321", 10)

// NewSpongeGoldilocks creates a variable-width sponge over the Goldilocks field.
// This is the STARK-friendly entry point for ci-poseidon.
func NewSpongeGoldilocks(mode SpongeMode) *SpongeState {
	return NewSponge(GoldilocksField, mode)
}

// ── First 512 primes (shared with ci-sha4096) ─────────────────────────────────

// primes holds the first 512 prime numbers, matching the prime sequence used
// in ci-sha4096's K-constant derivation. This ensures the field-reduced
// constants are the field-native counterparts of the original uint64 constants.
var primes = func() []int64 {
	out := make([]int64, 0, 512)
	sieve := make([]bool, 3700)
	for i := range sieve {
		sieve[i] = true
	}
	sieve[0], sieve[1] = false, false
	for i := 2; i < len(sieve); i++ {
		if sieve[i] {
			out = append(out, int64(i))
			if len(out) == 512 {
				break
			}
			for j := i * i; j < len(sieve); j += i {
				sieve[j] = false
			}
		}
	}
	return out
}()

// ── K-constant generation ─────────────────────────────────────────────────────

// KConstantField computes a single K-constant reduced modulo primeOrder.
//
// The formula mirrors ci-sha4096 exactly, but instead of floor-dividing to
// a uint64, we compute the exact modular inverse form:
//
//	K[i] = (85 × prime[i] × 2^64) × inv(27 × (prime[i]+1))  mod p
//
// This is the field-native representation of the same rational constant
// 85/27, scaled by prime[i]/(prime[i]+1).
func KConstantField(i int, primeOrder *big.Int) *big.Int {
	if i < 0 || i >= len(primes) {
		panic(fmt.Sprintf("ci-poseidon: KConstantField index %d out of range [0, %d)", i, len(primes)))
	}

	p := big.NewInt(primes[i])
	shift64 := new(big.Int).Lsh(big.NewInt(1), 64)

	// numerator = 85 * prime * 2^64
	num := new(big.Int).Mul(big.NewInt(85), p)
	num.Mul(num, shift64)
	num.Mod(num, primeOrder)

	// denominator = 27 * (prime + 1)
	denom := new(big.Int).Add(p, big.NewInt(1))
	denom.Mul(big.NewInt(27), denom)
	denomInv := new(big.Int).ModInverse(denom, primeOrder)
	if denomInv == nil {
		panic(fmt.Sprintf("ci-poseidon: modular inverse undefined for prime index %d (field prime may be composite)", i))
	}

	result := new(big.Int).Mul(num, denomInv)
	result.Mod(result, primeOrder)
	return result
}

// GenerateKConstants returns `count` K-constants reduced modulo primeOrder.
// count must be in [1, 512].
func GenerateKConstants(count int, primeOrder *big.Int) []*big.Int {
	if count < 1 || count > 512 {
		panic(fmt.Sprintf("ci-poseidon: count %d out of range [1, 512]", count))
	}
	out := make([]*big.Int, count)
	for i := 0; i < count; i++ {
		out[i] = KConstantField(i, primeOrder)
	}
	return out
}

// ── R-constant generation ─────────────────────────────────────────────────────

// RConstantField packs a resonance matrix entry into a single field element.
//
// The packing mirrors ci-sha4096's packRConstant but reduces modulo primeOrder
// instead of storing the raw uint64. The 17-bit rotation is preserved —
// it is coprime to 64, giving full bit coverage before cycling.
//
//	combined = (tHz10 << 48) | (nm10 << 32) | (neighborX << 16) | neighborY
//	rotated  = combined >>> 17  (rotate right)
//	R[i]     = (combined XOR rotated) mod p
func RConstantField(tHz10, nm10, neighborX, neighborY uint16, primeOrder *big.Int) *big.Int {
	combined := (uint64(tHz10) << 48) |
		(uint64(nm10) << 32) |
		(uint64(neighborX) << 16) |
		uint64(neighborY)

	rotated := (combined >> 17) | (combined << 47)
	raw := combined ^ rotated

	return new(big.Int).Mod(new(big.Int).SetUint64(raw), primeOrder)
}

// ── Poseidon2 permutation stub ────────────────────────────────────────────────

// Poseidon2Params holds the parameters for a Poseidon2 permutation instance.
type Poseidon2Params struct {
	Width          int        // state width t (number of field elements)
	FullRounds     int        // rf — full S-box rounds (applied to all elements)
	PartialRounds  int        // rp — partial S-box rounds (applied to one element)
	RoundConstants []*big.Int // precomputed from GenerateKConstants
	FieldPrime     *big.Int
}

// NewPoseidon2Params creates a Poseidon2Params instance with ci-derived constants.
// Standard BN254 parameters: width=3, fullRounds=8, partialRounds=56.
func NewPoseidon2Params(width, fullRounds, partialRounds int, fieldPrime *big.Int) *Poseidon2Params {
	totalConstants := (fullRounds + partialRounds) * width
	if totalConstants > 512 {
		totalConstants = 512
	}
	return &Poseidon2Params{
		Width:          width,
		FullRounds:     fullRounds,
		PartialRounds:  partialRounds,
		RoundConstants: GenerateKConstants(totalConstants, fieldPrime),
		FieldPrime:     fieldPrime,
	}
}

// SBox applies the x^5 S-box used in Poseidon2 over the field.
// x^5 mod p is the standard choice for BN254 and BLS12-381.
func SBox(x, p *big.Int) *big.Int {
	x2 := new(big.Int).Mul(x, x)
	x2.Mod(x2, p)
	x4 := new(big.Int).Mul(x2, x2)
	x4.Mod(x4, p)
	x5 := new(big.Int).Mul(x4, x)
	x5.Mod(x5, p)
	return x5
}

// AddRoundConstant adds a round constant to a state element mod p.
func AddRoundConstant(x, c, p *big.Int) *big.Int {
	result := new(big.Int).Add(x, c)
	result.Mod(result, p)
	return result
}

// Permute runs the Poseidon2 permutation on the given state (in-place).
// This is a reference implementation — not optimised for production.
//
// TODO: implement full MDS matrix multiplication for production use.
// Currently applies only round constants and S-boxes as a structural skeleton.
func (params *Poseidon2Params) Permute(state []*big.Int) []*big.Int {
	if len(state) != params.Width {
		panic(fmt.Sprintf("ci-poseidon: state width %d does not match params width %d",
			len(state), params.Width))
	}

	p := params.FieldPrime
	rc := params.RoundConstants
	rcIdx := 0

	// Copy state to avoid mutating the input
	s := make([]*big.Int, params.Width)
	for i, v := range state {
		s[i] = new(big.Int).Set(v)
	}

	totalRounds := params.FullRounds + params.PartialRounds

	for round := 0; round < totalRounds; round++ {
		// Add round constants
		for i := 0; i < params.Width; i++ {
			if rcIdx < len(rc) {
				s[i] = AddRoundConstant(s[i], rc[rcIdx], p)
				rcIdx++
			}
		}

		// S-box layer
		if round < params.FullRounds/2 || round >= totalRounds-params.FullRounds/2 {
			// Full round — S-box on all elements
			for i := 0; i < params.Width; i++ {
				s[i] = SBox(s[i], p)
			}
		} else {
			// Partial round — S-box on first element only
			s[0] = SBox(s[0], p)
		}

		// TODO: MDS matrix multiplication
		// A proper MDS matrix (maximum distance separable) is required for
		// full diffusion. The matrix should be derived from the ci constant
		// structure — see RESEARCH/zk-friendly-extension.md for open questions
		// around using the 19/40 square symmetry split to inform MDS design.
	}

	return s
}

// ── Utility ───────────────────────────────────────────────────────────────────

// FieldElementHex returns the hex representation of a field element,
// zero-padded to 64 characters (256 bits).
func FieldElementHex(x *big.Int) string {
	return fmt.Sprintf("%064x", x)
}
