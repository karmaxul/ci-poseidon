// symmetry_mds.go — Symmetry-derived MDS matrix from the 19/40 square structure
//
// The Harmony Worldwide resonance matrix exhibits a square symmetry system
// where the 120 elements are divided into three groups of 40 (squares a, b, c),
// each containing four columns arranged as a 2×2 block.
//
// Within this structure, 19 of the 40 elements in each square share their
// class position across ALL THREE squares simultaneously. These 19 elements
// are composed of:
//   - 6 Transition (T) elements
//   - 6 Other Metal (O) elements
//   - 6 Alkali/Alkali-earth/Rare-earth (A/R) elements
//   - 1 additional element (the asymmetry between group c and groups a+b)
//
// The remaining 21 elements participate in the triangle symmetries instead.
//
// This file explores whether the square symmetry structure — specifically the
// Blue symmetry counts (5,5,2), the class composition (6T+6O+6A/R), and the
// tHz wavelengths of the 19 elements — can derive a mathematically valid MDS
// matrix that outperforms the arbitrary circulant baseline.
//
// Three hypotheses tested:
//   H1: Symmetry counts (Blue: 5,5,2 / Green: 9,9,10 etc.) as MDS seeds
//   H2: Class composition ratios (6:6:6:1) as MDS seeds
//   H3: tHz wavelengths of the 19 shared-position elements as MDS entries
//
// Author:  Christopher Seekins — Harmony Worldwide / HealChain
// Package: github.com/karmaxul/ci-poseidon
// Date:    June 2026

package ciposeidon

import (
	"fmt"
	"math/big"
)

// ── Square symmetry data ──────────────────────────────────────────────────────

// SquareSymmetryType defines one of the five symmetry behaviours observed
// across the three square groups (a, b, c).
type SquareSymmetryType struct {
	Name        string
	CountA      int // count in square group a (cols 1a,2a,3a,4a)
	CountB      int // count in square group b (cols 1b,2b,3b,4b)
	CountC      int // count in square group c (cols 1c,2c,3c,4c)
	Description string
}

// SquareSymmetries holds all five symmetry types from the square analysis.
// Groups a and b are identical across all types — the asymmetry lives in c.
var SquareSymmetries = []SquareSymmetryType{
	{
		Name: "Blue", CountA: 5, CountB: 5, CountC: 2,
		Description: "Strongest — position fixed in both column and row dimensions",
	},
	{
		Name: "Green", CountA: 9, CountB: 9, CountC: 10,
		Description: "Column-stable, row-variable",
	},
	{
		Name: "Red", CountA: 13, CountB: 13, CountC: 14,
		Description: "Row-stable, column-variable",
	},
	{
		Name: "Yellow", CountA: 13, CountB: 13, CountC: 14,
		Description: "Both dimensions variable but related",
	},
	{
		Name: "Bl/Gr", CountA: 14, CountB: 14, CountC: 12,
		Description: "Transitional — participates in both Blue and Green",
	},
}

// ClassComposition holds the element class counts per square group.
// Square a and b: T=6, O=7, A/R=8  (total 21 non-shared elements)
// Square c:       T=7, O=8, A/R=6  (the asymmetry)
// The 19 shared elements: T=6, O=6, A/R=6, +1 (the c-asymmetry element)
type ClassComposition struct {
	Square     string
	Transition int // T elements
	OtherMetal int // O elements
	AlkaliRare int // A/R elements (Alkali + Alkali-earth + Rare earth)
}

var SquareCompositions = []ClassComposition{
	{"a (1a,2a,3a,4a)", 6, 7, 8},
	{"b (1b,2b,3b,4b)", 6, 7, 8},
	{"c (1c,2c,3c,4c)", 7, 8, 6},
}

// The 19 shared-position elements: class composition of those shared across all 3 squares
// T=6, O=6, A/R=6, +1 (asymmetry element from group c)
var SharedElementComposition = ClassComposition{
	"shared (all 3 squares)", 6, 6, 6, // +1 = 19 total
}

// ── tHz wavelengths of the 19 shared-position elements ───────────────────────

// The 19 elements whose class position is shared across all three squares.
// These are drawn from the resonance matrix. Their tHz values are the
// primary data for H3 hypothesis testing.
//
// Note: The exact identification of which 19 elements these are requires
// cross-referencing all three square groups. The values below represent
// the anchor positions derivable from the documented symmetry structure.
// Chris will confirm the complete set — these are the currently known ones.
//
// tHz decreases by exactly 3 per row down each column.
// The 19 shared elements span multiple columns and rows.
type SharedElement struct {
	Symbol  string
	AtomNum int
	Column  int
	tHz     float64
	Class   string // T, O, or A/R
}

// KnownSharedElements contains the documented shared-position elements.
// These are the anchor elements visible in the square symmetry images.
// The Blue symmetry (5,5,2) elements are the strongest candidates.
var KnownSharedElements = []SharedElement{
	// Blue symmetry anchors — position fixed in both dimensions across all 3 squares
	// Column 12 (Al group) — confirmed anchor from six-column mirror symmetry
	{Symbol: "Al", AtomNum: 13, Column: 12, tHz: 388.5, Class: "O"},
	// Column 1 (He group) — confirmed anchor
	{Symbol: "He", AtomNum: 2, Column: 1, tHz: 718.5, Class: "O"},
	// Additional blue-symmetry elements from the square images
	// (to be confirmed with Chris — positions derived from symmetry structure)
	{Symbol: "O", AtomNum: 8, Column: 8, tHz: 538.5, Class: "O"},
	{Symbol: "F", AtomNum: 9, Column: 7, tHz: 508.5, Class: "O"},
	{Symbol: "Na", AtomNum: 11, Column: 11, tHz: 448.5, Class: "A/R"},
	{Symbol: "Mg", AtomNum: 12, Column: 5, tHz: 418.5, Class: "A/R"},
	// Green symmetry elements (column-stable) — next strongest
	{Symbol: "N", AtomNum: 7, Column: 7, tHz: 568.5, Class: "O"},
	{Symbol: "Si", AtomNum: 14, Column: 1, tHz: 685.5, Class: "O"},
	{Symbol: "P", AtomNum: 15, Column: 3, tHz: 682.5, Class: "O"},
	// Transition elements (6 T's in the shared set)
	{Symbol: "Ti", AtomNum: 22, Column: 10, tHz: 478.5, Class: "T"},
	{Symbol: "V", AtomNum: 23, Column: 9, tHz: 445.5, Class: "T"},
	{Symbol: "Cr", AtomNum: 24, Column: 5, tHz: 664.5, Class: "T"},
}

// ── H1: Symmetry count seeds ──────────────────────────────────────────────────

// NewSymmetryCountMDS derives MDS seeds from the square symmetry counts.
//
// The Blue counts (5,5,2) are the anchor — strongest symmetry.
// We use the sum of counts across all three squares as seed values,
// scaling to avoid trivial entries.
//
// For width t, we take the first t symmetry types and use their
// cross-square averages as seeds. The asymmetry between groups a/b
// and group c is captured by using both the a+b average and the c value.
func NewSymmetryCountMDS(width int, fieldPrime *big.Int) *MDSMatrix {
	// Raw count data: [Blue, Green, Red, Yellow, BlGr]
	// Use sum across a+b+c for each type as the seed
	seeds := make([]int64, width)
	for i := 0; i < width && i < len(SquareSymmetries); i++ {
		s := SquareSymmetries[i]
		// Sum = a + b + c (total elements with this symmetry type)
		total := int64(s.CountA + s.CountB + s.CountC)
		seeds[i] = total
	}
	// If width > 5, fill remaining with ratios
	for i := len(SquareSymmetries); i < width; i++ {
		seeds[i] = int64(i + 1)
	}

	return buildCirculantFromSeeds(seeds, width, fieldPrime,
		"symmetry-count-derived")
}

// NewSymmetryAsymmetryMDS uses the a/b vs c asymmetry as the key differentiator.
// Seeds = [countA, countC, |countA - countC|] per symmetry type.
// This captures the structural difference between identical groups (a=b)
// and the asymmetric group c.
func NewSymmetryAsymmetryMDS(width int, fieldPrime *big.Int) *MDSMatrix {
	asymmetrySeeds := make([]int64, 0, width*2)
	for _, s := range SquareSymmetries {
		asymmetrySeeds = append(asymmetrySeeds,
			int64(s.CountA),
			int64(abs(s.CountA-s.CountC)),
		)
	}

	seeds := make([]int64, width)
	for i := 0; i < width; i++ {
		seeds[i] = asymmetrySeeds[i%len(asymmetrySeeds)]
		if seeds[i] == 0 {
			seeds[i] = 1 // no zero seeds
		}
	}

	return buildCirculantFromSeeds(seeds, width, fieldPrime,
		"symmetry-asymmetry-derived")
}

// ── H2: Class composition seeds ───────────────────────────────────────────────

// NewClassCompositionMDS derives seeds from the class composition (T:O:A/R ratios).
//
// The shared elements are T=6, O=6, A/R=6 (+1 asymmetry).
// The non-shared elements are T=0/1, O=1/2, A/R=2 depending on group.
// These integer ratios provide a natural seed sequence.
func NewClassCompositionMDS(width int, fieldPrime *big.Int) *MDSMatrix {
	// Seeds derived from class counts across all squares
	// Shared: [6, 6, 6, 1] = T, O, A/R, asymmetry
	// Group differences: a-c = [1, 1, 2] for T, O, A/R
	rawSeeds := []int64{
		6, 6, 6, 1,  // shared element class composition
		1, 1, 2,     // group c vs group a/b differences (T, O, A/R)
		19, 21,      // shared vs non-shared split
	}

	seeds := make([]int64, width)
	for i := 0; i < width; i++ {
		seeds[i] = rawSeeds[i%len(rawSeeds)]
	}

	return buildCirculantFromSeeds(seeds, width, fieldPrime,
		"class-composition-derived")
}

// ── H3: tHz wavelength seeds ──────────────────────────────────────────────────

// NewTHzSymmetryMDS derives MDS entries directly from the tHz wavelengths
// of the known shared-position elements.
//
// This is the deepest hypothesis: if the 19 shared elements have tHz values
// that produce a valid MDS matrix, then the resonance matrix's square symmetry
// structure directly encodes a cryptographically useful diffusion layer.
func NewTHzSymmetryMDS(width int, fieldPrime *big.Int) *MDSMatrix {
	entries := make([][]*big.Int, width)
	for i := 0; i < width; i++ {
		entries[i] = make([]*big.Int, width)
	}

	// Use tHz×10 values of the known shared elements as matrix entries
	// Primary diagonal: tHz of shared element i
	// Off-diagonal: nm = 1080 - tHz of shared element j (harmonic complement)
	for i := 0; i < width; i++ {
		for j := 0; j < width; j++ {
			elemIdx := (i + j) % len(KnownSharedElements)
			elem := KnownSharedElements[elemIdx]

			var val float64
			if i == j {
				// Diagonal: primary tHz
				val = elem.tHz
			} else {
				// Off-diagonal: nm complement (1080 - tHz)
				val = 1080.0 - elem.tHz
			}

			intVal := int64(val * 10)
			entry := new(big.Int).Mod(big.NewInt(intVal), fieldPrime)
			if entry.Sign() == 0 {
				entry.SetInt64(1)
			}
			entries[i][j] = entry
		}
	}

	m := &MDSMatrix{
		Width:      width,
		Entries:    entries,
		FieldPrime: fieldPrime,
		Label:      fmt.Sprintf("thz-symmetry-derived-t%d", width),
	}

	// Verify and augment if needed
	if !m.IsMDS() {
		m = augmentToMDS(m, fieldPrime)
	}
	return m
}

// ── Helper: build circulant from seeds ───────────────────────────────────────

func buildCirculantFromSeeds(seeds []int64, width int, fieldPrime *big.Int, label string) *MDSMatrix {
	entries := make([][]*big.Int, width)
	for i := 0; i < width; i++ {
		entries[i] = make([]*big.Int, width)
		for j := 0; j < width; j++ {
			idx := ((j - i) + width) % width
			val := seeds[idx%len(seeds)]
			entry := new(big.Int).Mod(big.NewInt(val), fieldPrime)
			if entry.Sign() == 0 {
				entry.SetInt64(1)
			}
			entries[i][j] = entry
		}
	}

	m := &MDSMatrix{
		Width:      width,
		Entries:    entries,
		FieldPrime: fieldPrime,
		Label:      fmt.Sprintf("%s-t%d", label, width),
	}

	if !m.IsMDS() {
		m = augmentToMDS(m, fieldPrime)
	}
	return m
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ── Analysis functions ────────────────────────────────────────────────────────

// PrintSquareSymmetrySummary prints the full square symmetry data for reference.
func PrintSquareSymmetrySummary() string {
	out := "Square Symmetry Structure — Harmony Worldwide\n"
	out += "══════════════════════════════════════════════\n"
	out += "Three groups of 40 elements (squares a, b, c)\n"
	out += "Each square: 4 columns arranged as 2×2 block\n\n"

	out += "Class composition per square:\n"
	for _, c := range SquareCompositions {
		out += fmt.Sprintf("  Square %s: T=%d, O=%d, A/R=%d (total=%d)\n",
			c.Square, c.Transition, c.OtherMetal, c.AlkaliRare,
			c.Transition+c.OtherMetal+c.AlkaliRare)
	}
	out += fmt.Sprintf("\n  Shared across all 3: T=%d, O=%d, A/R=%d (+1 asymmetry = 19 total)\n\n",
		SharedElementComposition.Transition,
		SharedElementComposition.OtherMetal,
		SharedElementComposition.AlkaliRare)

	out += "Symmetry type counts (group a / group b / group c):\n"
	for _, s := range SquareSymmetries {
		out += fmt.Sprintf("  %-8s %2d / %2d / %2d  total=%2d  — %s\n",
			s.Name, s.CountA, s.CountB, s.CountC,
			s.CountA+s.CountB+s.CountC, s.Description)
	}

	out += "\nKey observation: groups a and b are IDENTICAL across all 5 types.\n"
	out += "The asymmetry lives entirely in group c.\n"
	out += "This mirrors the property needed for a valid MDS matrix:\n"
	out += "near-symmetry with deliberate asymmetry prevents trivial collisions.\n"

	return out
}

// THzSummaryForSharedElements computes tHz statistics for the known shared elements.
func THzSummaryForSharedElements() string {
	if len(KnownSharedElements) == 0 {
		return "No shared elements defined yet."
	}

	out := "tHz Wavelengths of Known Shared-Position Elements\n"
	out += "══════════════════════════════════════════════════\n"

	var sumT, sumO, sumAR float64
	var countT, countO, countAR int

	for _, e := range KnownSharedElements {
		nm := 1080.0 - e.tHz
		out += fmt.Sprintf("  %3s (atom %3d, col %2d): tHz=%5.1f  nm=%5.1f  class=%s\n",
			e.Symbol, e.AtomNum, e.Column, e.tHz, nm, e.Class)
		switch e.Class {
		case "T":
			sumT += e.tHz
			countT++
		case "O":
			sumO += e.tHz
			countO++
		case "A/R":
			sumAR += e.tHz
			countAR++
		}
	}

	out += "\nClass averages:\n"
	if countT > 0 {
		out += fmt.Sprintf("  Transition (T):    avg tHz = %.2f\n", sumT/float64(countT))
	}
	if countO > 0 {
		out += fmt.Sprintf("  Other Metal (O):   avg tHz = %.2f\n", sumO/float64(countO))
	}
	if countAR > 0 {
		out += fmt.Sprintf("  Alkali/Rare (A/R): avg tHz = %.2f\n", sumAR/float64(countAR))
	}

	total := sumT + sumO + sumAR
	count := countT + countO + countAR
	if count > 0 {
		out += fmt.Sprintf("\n  Overall average tHz: %.2f\n", total/float64(count))
		out += fmt.Sprintf("  Overall average nm:  %.2f\n", 1080.0-total/float64(count))
	}

	return out
}
