// sponge.go — Variable-width sponge with structured state transitions
//
// This is the core novelty of ci-Poseidon: a sponge construction that
// starts at t=2 and expands through t=3 → t=4 → t=6 as needed, with
// every transition governed by the harmonic structure of the resonance
// matrix rather than arbitrary thresholds.
//
// The "breathing" behaviour:
//
//   - The sponge begins absorbing input at t=2 (minimal, fast)
//   - After each permutation, a diffusion score is computed
//   - If the score falls below the tHz expansion threshold, the state
//     expands to the next width
//   - If the score rises above the nm contraction threshold, the state
//     contracts back toward t=2
//   - tHz and nm are complementary (tHz + nm = 1080) so the two
//     thresholds are always in harmonic balance
//   - Width transitions inject new K-constants derived from the resonance
//     matrix at the expansion point — the new state elements are not
//     zero-padded but harmonically seeded
//
// Width progression: t=2 → t=3 → t=4 → t=6
// (contractions follow the reverse path)
//
// Author:  Christopher Seekins — Harmony Worldwide / HealChain
// Package: github.com/karmaxul/ci-poseidon
// Date:    June 2026

package ciposeidon

import (
	"fmt"
	"math/big"
)

// ── Width ladder ──────────────────────────────────────────────────────────────

// widthLadder defines the ordered sequence of supported state widths.
// The sponge climbs and descends this ladder based on diffusion scores.
var widthLadder = []int{2, 3, 4, 6}

// widthIndex returns the position of w in the width ladder, or -1 if not found.
func widthIndex(w int) int {
	for i, v := range widthLadder {
		if v == w {
			return i
		}
	}
	return -1
}

// nextWidth returns the next wider state, or the same if already at maximum.
func nextWidth(w int) int {
	idx := widthIndex(w)
	if idx < 0 || idx >= len(widthLadder)-1 {
		return w
	}
	return widthLadder[idx+1]
}

// prevWidth returns the next narrower state, or the same if already at minimum.
func prevWidth(w int) int {
	idx := widthIndex(w)
	if idx <= 0 {
		return w
	}
	return widthLadder[idx-1]
}

// ── Resonance thresholds ──────────────────────────────────────────────────────

// ResonanceThresholds holds the expansion and contraction thresholds for a
// given state width, derived from the resonance matrix anchor values.
//
// expandBelow:   if diffusion score < this value, expand to next width
//               derived from tHz of the width's anchor element (×10)
//
// contractAbove: if diffusion score > this value, contract to previous width
//               derived from nm of the width's anchor element (×10)
//               since nm = 1080 - tHz, contractAbove > expandBelow always
//               — the two thresholds are in harmonic balance by construction
type ResonanceThresholds struct {
	Width         int
	ExpandBelow   uint64 // tHz × 10 — low diffusion triggers expansion
	ContractAbove uint64 // nm × 10  — high diffusion allows contraction
}

// thresholds maps each width to its resonance-derived thresholds.
// Anchor elements match the MDS seed entries for consistency:
//   t=2: Al (tHz=388.5 → 3885, nm=691.5 → 6915)  — col 12 anchor
//   t=3: O  (tHz=538.5 → 5385, nm=541.5 → 5415)  — col 8, near midpoint
//   t=4: F  (tHz=508.5 → 5085, nm=571.5 → 5715)  — col 7
//   t=6: Na (tHz=448.5 → 4485, nm=631.5 → 6315)  — col 11
var thresholds = map[int]ResonanceThresholds{
	2: {Width: 2, ExpandBelow: 3885, ContractAbove: 6915},
	3: {Width: 3, ExpandBelow: 5385, ContractAbove: 5415},
	4: {Width: 4, ExpandBelow: 5085, ContractAbove: 5715},
	6: {Width: 6, ExpandBelow: 4485, ContractAbove: 6315},
}

// ── Sponge state ──────────────────────────────────────────────────────────────

// SpongeMode controls whether the sponge uses circulant or ci-derived MDS.
type SpongeMode int

const (
	ModeCirculant SpongeMode = iota // baseline / control
	ModeCiDerived                   // experimental — ci-derived MDS
)

func (m SpongeMode) String() string {
	if m == ModeCirculant {
		return "circulant-baseline"
	}
	return "ci-derived"
}

// SpongeState holds the full runtime state of the variable-width sponge.
type SpongeState struct {
	State        []*big.Int // current state vector (length = CurrentWidth)
	CurrentWidth int
	FieldPrime   *big.Int
	Mode         SpongeMode
	MDS          *MDSMatrix // current MDS matrix (matches CurrentWidth)

	// Diagnostics
	ExpandCount  int   // number of times the sponge expanded
	ContractCount int  // number of times the sponge contracted
	PermCount    int   // total permutations applied
	WidthHistory []int // width at each permutation step
}

// NewSponge creates a new variable-width sponge starting at t=2.
func NewSponge(fieldPrime *big.Int, mode SpongeMode) *SpongeState {
	initialWidth := 2
	s := &SpongeState{
		CurrentWidth: initialWidth,
		FieldPrime:   fieldPrime,
		Mode:         mode,
		WidthHistory: []int{},
	}
	s.State = s.seedState(initialWidth, 0)
	s.MDS = s.newMDS(initialWidth)
	return s
}

// ── Core sponge operations ────────────────────────────────────────────────────

// Absorb ingests a field element into the sponge state.
// The element is added to the first state position, then the sponge permutes.
// If the diffusion score after permutation is too low, the state expands.
func (s *SpongeState) Absorb(input *big.Int) {
	// XOR input into first state element (standard sponge absorption)
	s.State[0] = new(big.Int).Add(s.State[0], input)
	s.State[0].Mod(s.State[0], s.FieldPrime)

	// Permute
	s.permute()

	// Check diffusion and adapt width
	score := s.diffusionScore()
	thresh := thresholds[s.CurrentWidth]

	if score < thresh.ExpandBelow && s.CurrentWidth < widthLadder[len(widthLadder)-1] {
		s.expand()
	} else if score > thresh.ContractAbove && s.CurrentWidth > widthLadder[0] {
		s.contract()
	}
}

// AbsorbAll ingests a slice of field elements, one at a time.
func (s *SpongeState) AbsorbAll(inputs []*big.Int) {
	for _, v := range inputs {
		s.Absorb(v)
	}
}

// Squeeze extracts `count` field elements from the sponge.
// Each squeeze permutes the state and extracts the first element.
func (s *SpongeState) Squeeze(count int) []*big.Int {
	out := make([]*big.Int, count)
	for i := 0; i < count; i++ {
		s.permute()
		out[i] = new(big.Int).Set(s.State[0])
	}
	return out
}

// Hash is a convenience function: absorb all inputs, squeeze `outLen` elements.
func (s *SpongeState) Hash(inputs []*big.Int, outLen int) []*big.Int {
	s.AbsorbAll(inputs)
	return s.Squeeze(outLen)
}

// ── Internal permutation ──────────────────────────────────────────────────────

// permute delegates to the canonical ApplyPermutation engine in permutation.go.
// This eliminates the previous duplicated round logic and ensures the sponge
// always uses the properly tuned HADES structure with correct round parameters.
func (s *SpongeState) permute() {
	rc := NewRoundConstants(s.CurrentWidth, s.FieldPrime)

	// Copy state so ApplyPermutation works on a fresh slice
	st := make([]*big.Int, s.CurrentWidth)
	for i, v := range s.State {
		st[i] = new(big.Int).Set(v)
	}

	ApplyPermutation(st, rc, s.MDS)

	s.State = st
	s.PermCount++
	s.WidthHistory = append(s.WidthHistory, s.CurrentWidth)
}

// ── Diffusion scoring ─────────────────────────────────────────────────────────

// diffusionScore measures how well-distributed the current state is.
//
// Score = sum of pairwise differences between state elements, normalised
// to a uint64 range. A low score means elements are clustered (poor diffusion),
// a high score means elements are spread (good diffusion).
//
// The thresholds (tHz×10 and nm×10) are in the same uint64 range, so the
// comparison is direct and meaningful.
func (s *SpongeState) diffusionScore() uint64 {
	if len(s.State) < 2 {
		return 0
	}
	p := s.FieldPrime

	// Compute sum of all pairwise absolute differences mod p,
	// then take the low 64 bits as a proxy score.
	total := big.NewInt(0)
	count := 0
	for i := 0; i < len(s.State); i++ {
		for j := i + 1; j < len(s.State); j++ {
			diff := new(big.Int).Sub(s.State[i], s.State[j])
			diff.Abs(diff)
			diff.Mod(diff, p)
			total.Add(total, diff)
			count++
		}
	}
	if count > 0 {
		total.Div(total, big.NewInt(int64(count)))
	}

	// Return low 64 bits scaled to threshold range
	// Scale: divide by (p / 10000) to bring into the ~0–10000 range
	// where our thresholds (3885–6915) live
	scaler := new(big.Int).Div(p, big.NewInt(10000))
	if scaler.Sign() == 0 {
		return 0
	}
	scaled := new(big.Int).Div(total, scaler)
	if !scaled.IsInt64() {
		return 9999
	}
	v := scaled.Int64()
	if v < 0 {
		return 0
	}
	if v > 9999 {
		return 9999
	}
	return uint64(v)
}

// ── State transitions ─────────────────────────────────────────────────────────

// expand grows the state from width w to nextWidth(w).
// The new state elements are harmonically seeded from K-constants at the
// expansion point — not zero-padded, which would create weak initial states.
func (s *SpongeState) expand() {
	newWidth := nextWidth(s.CurrentWidth)
	if newWidth == s.CurrentWidth {
		return
	}

	// Grow state: keep existing elements, seed new ones from K-constants
	newState := make([]*big.Int, newWidth)
	copy(newState, s.State)
	for i := s.CurrentWidth; i < newWidth; i++ {
		// Seed new elements from K-constants offset by current perm count
		// so each expansion produces distinct seeding
		newState[i] = KConstantField((s.PermCount+i)%512, s.FieldPrime)
	}

	s.State = newState
	s.CurrentWidth = newWidth
	s.MDS = s.newMDS(newWidth)
	s.ExpandCount++
}

// contract shrinks the state from width w to prevWidth(w).
// The state is folded: each dropped element is XOR-added back into the
// remaining state elements so no information is discarded.
func (s *SpongeState) contract() {
	newWidth := prevWidth(s.CurrentWidth)
	if newWidth == s.CurrentWidth {
		return
	}

	p := s.FieldPrime

	// Fold dropped elements back into the retained state
	newState := make([]*big.Int, newWidth)
	for i := 0; i < newWidth; i++ {
		newState[i] = new(big.Int).Set(s.State[i])
	}
	for i := newWidth; i < s.CurrentWidth; i++ {
		idx := i % newWidth
		newState[idx] = new(big.Int).Add(newState[idx], s.State[i])
		newState[idx].Mod(newState[idx], p)
	}

	s.State = newState
	s.CurrentWidth = newWidth
	s.MDS = s.newMDS(newWidth)
	s.ContractCount++
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// seedState creates an initial state vector of the given width, seeded from
// K-constants at the given offset. Never zero-initialised.
func (s *SpongeState) seedState(width, offset int) []*big.Int {
	state := make([]*big.Int, width)
	for i := 0; i < width; i++ {
		state[i] = KConstantField((offset+i)%512, s.FieldPrime)
	}
	return state
}

// newMDS constructs the appropriate MDS matrix for the given width and mode.
func (s *SpongeState) newMDS(width int) *MDSMatrix {
	if s.Mode == ModeCiDerived {
		return NewCiDerivedMDS(width, s.FieldPrime)
	}
	return NewCirculantMDS(width, s.FieldPrime)
}

// DiagnosticsString returns a human-readable summary of sponge behaviour.
func (s *SpongeState) DiagnosticsString() string {
	return fmt.Sprintf(
		"SpongeState{mode=%s, width=%d, perms=%d, expands=%d, contracts=%d}",
		s.Mode, s.CurrentWidth, s.PermCount, s.ExpandCount, s.ContractCount,
	)
}
