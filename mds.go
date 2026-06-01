// mds.go — MDS matrix construction for ci-Poseidon
//
// Two construction methods are provided for each supported width:
//
//   1. Circulant (baseline/control) — standard cryptographic construction
//      using well-known MDS-safe values. This is the control in our experiment.
//
//   2. Ci-derived (experimental) — matrix entries derived from the Harmony
//      Worldwide resonance matrix. tHz values govern the primary layer,
//      nm values (= 1080 - tHz, always) govern the complementary layer.
//      The two are in perfect harmonic balance by construction.
//
// Supported widths: t=2, t=3, t=4, t=6
//
// The variable-width sponge (see sponge.go) uses these matrices as the
// diffusion layer at each width level. When the state expands from t=2 to
// t=3 to t=4 to t=6, the MDS matrix transitions with it — each width's
// matrix is derived from the same harmonic source, so transitions are
// mathematically principled rather than arbitrary.
//
// MDS property: a t×t matrix M is MDS (Maximum Distance Separable) if
// every square submatrix is invertible. This guarantees that any change
// in k input elements affects all output elements — full diffusion.
//
// Author:  Christopher Seekins — Harmony Worldwide / HealChain
// Package: github.com/karmaxul/ci-poseidon
// Date:    June 2026

package ciposeidon

import (
	"fmt"
	"math/big"
)

// ── MDS Matrix type ───────────────────────────────────────────────────────────

// MDSMatrix is a t×t matrix of field elements used as the diffusion layer
// in the Poseidon2 permutation.
type MDSMatrix struct {
	Width      int
	Entries    [][]*big.Int // Entries[row][col]
	FieldPrime *big.Int
	Label      string // "circulant-baseline" or "ci-derived"
}

// Apply multiplies the state vector by the MDS matrix (mod p).
// state must have length == matrix width.
func (m *MDSMatrix) Apply(state []*big.Int) []*big.Int {
	if len(state) != m.Width {
		panic(fmt.Sprintf("mds: state width %d != matrix width %d", len(state), m.Width))
	}
	p := m.FieldPrime
	out := make([]*big.Int, m.Width)
	for i := 0; i < m.Width; i++ {
		sum := big.NewInt(0)
		for j := 0; j < m.Width; j++ {
			term := new(big.Int).Mul(m.Entries[i][j], state[j])
			term.Mod(term, p)
			sum.Add(sum, term)
			sum.Mod(sum, p)
		}
		out[i] = sum
	}
	return out
}

// IsMDS verifies the MDS property by checking that every square submatrix
// has a non-zero determinant. For t ≤ 6 this is computationally feasible.
// Returns true if MDS, false otherwise.
func (m *MDSMatrix) IsMDS() bool {
	// For each possible subset of rows and columns of the same size k,
	// check that the k×k submatrix is invertible (det ≠ 0 mod p).
	t := m.Width
	for k := 1; k <= t; k++ {
		rowSets := combinations(t, k)
		colSets := combinations(t, k)
		for _, rows := range rowSets {
			for _, cols := range colSets {
				sub := extractSubmatrix(m.Entries, rows, cols)
				det := determinant(sub, m.FieldPrime)
				if det.Sign() == 0 {
					return false
				}
			}
		}
	}
	return true
}

// ── Circulant baseline construction ──────────────────────────────────────────

// circulantRow holds the seed values for the circulant MDS baseline at each width.
// These are well-known MDS-safe values from the Poseidon2 literature.
// Each subsequent row is the previous row shifted right by one position.
var circulantSeeds = map[int][]int64{
	2: {2, 1},
	3: {2, 1, 1},
	4: {5, 7, 1, 3},
	6: {10, 11, 13, 5, 2, 1},
}

// NewCirculantMDS constructs a circulant MDS matrix for the given width.
// The matrix is built from well-known baseline seed values.
func NewCirculantMDS(width int, fieldPrime *big.Int) *MDSMatrix {
	seeds, ok := circulantSeeds[width]
	if !ok {
		panic(fmt.Sprintf("mds: unsupported circulant width %d (supported: 2,3,4,6)", width))
	}

	entries := make([][]*big.Int, width)
	for i := 0; i < width; i++ {
		entries[i] = make([]*big.Int, width)
		for j := 0; j < width; j++ {
			// Circulant: entry[i][j] = seed[(j-i+width) % width]
			idx := ((j - i) + width) % width
			entries[i][j] = new(big.Int).Mod(
				big.NewInt(seeds[idx]),
				fieldPrime,
			)
		}
	}

	return &MDSMatrix{
		Width:      width,
		Entries:    entries,
		FieldPrime: fieldPrime,
		Label:      fmt.Sprintf("circulant-baseline-t%d", width),
	}
}

// ── Ci-derived construction ───────────────────────────────────────────────────

// resonanceEntry holds tHz and nm for a single resonance matrix element.
// nm = 1080 - tHz always — they are complementary by construction.
// This is the harmonic balance that governs the ci-derived MDS entries.
type resonanceEntry struct {
	tHz float64
	nm  float64 // always 1080 - tHz
}

// seedEntries provides the resonance matrix anchor values used to seed the
// ci-derived MDS matrices at each width.
//
// These are drawn from the column anchor positions in the Harmony Worldwide
// resonance matrix — the elements at the structural pivot of each symmetry group.
//
// Width 2: Al(col12, tHz=388.5) and He(col1, tHz=718.5) — the orange anchors
// Width 3: Al(388.5), He(718.5), O(538.5) — first element in col1 cycle
// Width 4: Al(388.5), He(718.5), O(538.5), F(508.5) — cols 12,1,8,7
// Width 6: Al(388.5), He(718.5), O(538.5), F(508.5), Na(448.5), Mg(418.5)
//          — the six anchor elements of the mirror symmetry structure
var seedEntries = map[int][]resonanceEntry{
	2: {
		{tHz: 388.5, nm: 691.5}, // Al — col 12 anchor
		{tHz: 718.5, nm: 361.5}, // He — col 1 anchor
	},
	3: {
		{tHz: 388.5, nm: 691.5}, // Al — col 12
		{tHz: 718.5, nm: 361.5}, // He — col 1
		{tHz: 538.5, nm: 541.5}, // O  — col 8 (nearest to midpoint 540)
	},
	4: {
		{tHz: 388.5, nm: 691.5}, // Al — col 12
		{tHz: 718.5, nm: 361.5}, // He — col 1
		{tHz: 538.5, nm: 541.5}, // O  — col 8
		{tHz: 508.5, nm: 571.5}, // F  — col 7
	},
	6: {
		{tHz: 388.5, nm: 691.5}, // Al — col 12
		{tHz: 718.5, nm: 361.5}, // He — col 1
		{tHz: 538.5, nm: 541.5}, // O  — col 8
		{tHz: 508.5, nm: 571.5}, // F  — col 7
		{tHz: 448.5, nm: 631.5}, // Na — col 11
		{tHz: 418.5, nm: 661.5}, // Mg — col 5
	},
}

// ciDerivedEntry computes a single MDS matrix entry from a resonance value.
//
// The entry is derived as:
//
//	floor(value × 10) mod p
//
// using ×10 to convert the .5 decimal values to exact integers (3885, 7185...)
// preserving full precision with no floating-point loss.
//
// The tHz and nm layers are complementary: tHz10 + nm10 = 10800 always.
// This means the two diagonals of the matrix sum to a constant — a structural
// property no arbitrary circulant matrix can claim.
func ciDerivedEntry(value float64, fieldPrime *big.Int) *big.Int {
	// Multiply by 10 to get exact integer, no floating point loss
	// e.g. 388.5 → 3885,  718.5 → 7185
	intVal := int64(value * 10)
	return new(big.Int).Mod(big.NewInt(intVal), fieldPrime)
}

// NewCiDerivedMDS constructs an MDS matrix whose entries are derived from
// the Harmony Worldwide resonance matrix anchor values.
//
// The primary diagonal uses tHz values; the complementary layer uses nm values.
// Since tHz + nm = 1080 for every element, the two layers are in perfect
// harmonic balance — a property intrinsic to the resonance matrix.
//
// If the derived matrix fails the MDS property (detected automatically),
// the entries are augmented by adding successive K-constants until MDS is
// achieved. This ensures we always get a valid MDS matrix while staying
// as close as possible to the resonance data.
func NewCiDerivedMDS(width int, fieldPrime *big.Int) *MDSMatrix {
	seeds, ok := seedEntries[width]
	if !ok {
		panic(fmt.Sprintf("mds: unsupported ci-derived width %d (supported: 2,3,4,6)", width))
	}

	entries := make([][]*big.Int, width)
	for i := 0; i < width; i++ {
		entries[i] = make([]*big.Int, width)
		for j := 0; j < width; j++ {
			// Primary layer: tHz of seed[(i+j) % width]
			// Complementary layer: nm of seed[(i+j+1) % width]
			// The two layers interleave across the matrix, encoding
			// both the tHz and nm harmonic relationships.
			primary := ciDerivedEntry(seeds[(i+j)%width].tHz, fieldPrime)
			complement := ciDerivedEntry(seeds[(i+j+1)%width].nm, fieldPrime)

			// Combine: primary + complement mod p
			entry := new(big.Int).Add(primary, complement)
			entry.Mod(entry, fieldPrime)

			// Ensure no zero entries (zero breaks MDS)
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
		Label:      fmt.Sprintf("ci-derived-t%d", width),
	}

	// Verify MDS property — augment with K-constants if needed
	if !m.IsMDS() {
		m = augmentToMDS(m, fieldPrime)
	}

	return m
}

// augmentToMDS adds successive K-constants to matrix entries until the MDS
// property is satisfied. Entries are perturbed row by row, wrapping around,
// until the matrix becomes MDS. This is a fallback — in practice the
// resonance-derived entries should be very close to MDS for small widths.
func augmentToMDS(m *MDSMatrix, fieldPrime *big.Int) *MDSMatrix {
	maxAttempts := 1024
	kIdx := 0
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if m.IsMDS() {
			return m
		}
		// Perturb entry [row][col] with next K-constant
		row := kIdx % m.Width
		col := (kIdx / m.Width) % m.Width
		k := KConstantField(kIdx%512, fieldPrime)
		m.Entries[row][col] = new(big.Int).Add(m.Entries[row][col], k)
		m.Entries[row][col].Mod(m.Entries[row][col], fieldPrime)
		// Ensure no zero entries
		if m.Entries[row][col].Sign() == 0 {
			m.Entries[row][col].SetInt64(1)
		}
		kIdx++
	}
	panic(fmt.Sprintf("mds: could not achieve MDS property after %d augmentation attempts", maxAttempts))
}

// ── Matrix utilities ──────────────────────────────────────────────────────────

// combinations returns all k-element subsets of {0, 1, ..., n-1}.
func combinations(n, k int) [][]int {
	if k == 0 {
		return [][]int{{}}
	}
	if k > n {
		return nil
	}
	result := [][]int{}
	var helper func(start int, current []int)
	helper = func(start int, current []int) {
		if len(current) == k {
			cp := make([]int, k)
			copy(cp, current)
			result = append(result, cp)
			return
		}
		for i := start; i < n; i++ {
			helper(i+1, append(current, i))
		}
	}
	helper(0, []int{})
	return result
}

// extractSubmatrix extracts the submatrix at the given row/col indices.
func extractSubmatrix(entries [][]*big.Int, rows, cols []int) [][]*big.Int {
	k := len(rows)
	sub := make([][]*big.Int, k)
	for i, r := range rows {
		sub[i] = make([]*big.Int, k)
		for j, c := range cols {
			sub[i][j] = new(big.Int).Set(entries[r][c])
		}
	}
	return sub
}

// determinant computes the determinant of a square matrix mod p
// using cofactor expansion. For t ≤ 6 this is fast enough.
func determinant(m [][]*big.Int, p *big.Int) *big.Int {
	n := len(m)
	if n == 1 {
		return new(big.Int).Mod(m[0][0], p)
	}
	if n == 2 {
		// ad - bc mod p
		ad := new(big.Int).Mul(m[0][0], m[1][1])
		bc := new(big.Int).Mul(m[0][1], m[1][0])
		det := new(big.Int).Sub(ad, bc)
		det.Mod(det, p)
		if det.Sign() < 0 {
			det.Add(det, p)
		}
		return det
	}

	det := big.NewInt(0)
	for col := 0; col < n; col++ {
		// Build minor by excluding row 0 and current col
		minor := make([][]*big.Int, n-1)
		for r := 1; r < n; r++ {
			minor[r-1] = make([]*big.Int, 0, n-1)
			for c := 0; c < n; c++ {
				if c != col {
					minor[r-1] = append(minor[r-1], m[r][c])
				}
			}
		}
		cofactor := determinant(minor, p)
		cofactor.Mul(m[0][col], cofactor)
		cofactor.Mod(cofactor, p)

		if col%2 == 0 {
			det.Add(det, cofactor)
		} else {
			det.Sub(det, cofactor)
		}
		det.Mod(det, p)
		if det.Sign() < 0 {
			det.Add(det, p)
		}
	}
	return det
}

// PrintMatrix prints the matrix entries as hex strings (for debugging/verification).
func (m *MDSMatrix) PrintMatrix() {
	fmt.Printf("MDS Matrix [%s] %d×%d\n", m.Label, m.Width, m.Width)
	for i, row := range m.Entries {
		fmt.Printf("  row %d: ", i)
		for _, v := range row {
			fmt.Printf("%s  ", FieldElementHex(v))
		}
		fmt.Println()
	}
}
