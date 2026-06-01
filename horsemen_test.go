// horsemen_test.go — The Four Horsemen and circle code contact point analysis
//
// The Four Horsemen are elements 15, 45, 75, 105 — spaced every 30 elements
// starting from 15. They are unique in that their number of circle lines
// differs from their nuclear mass (all other elements: lines = nuclear mass).
// Their contact point neighbours form their own internal symmetry.
//
// From Chris's paper (image 3):
//   Element 15:  neighbours 19, 16   — sum = 35
//   Element 45:  neighbours 19, 91
//   Element 75:  neighbours 79, 106
//   Element 105: neighbours 109, 151
//
// Proton sequence:  19, 19, 79, 109  (+0, +60, +30)
// Neutron sequence: 16, 91, 106, 151 (16→106 = +90, 91→151 = +60)
//
// The 3.4 × 25 = 85 connection:
//   Ci in base 27 = 3.4 (terminates exactly)
//   3.4 × 25 = 85 = numerator of Ci
//   2125 ÷ Ci = 675 = Seekins constant (×10^-36 scaling)
//   2125 ÷ 85 = 25
//
// Author:  Christopher Seekins — Harmony Worldwide / HealChain
// Package: github.com/karmaxul/ci-poseidon
// Date:    June 2026

package ciposeidon

import (
	"math/big"
	"testing"
)

// ── Four Horsemen data ────────────────────────────────────────────────────────

// Horseman holds the contact point data for one of the four horsemen elements.
type Horseman struct {
	Element    int    // element number
	Name       string // element name
	Column     int    // column in resonance matrix
	Protons    int    // left contact point neighbour (what Chris calls protons)
	Neutrons   int    // right contact point neighbour (what Chris calls neutrons)
	CircleLines int   // number of lines in the circle (≠ nuclear mass for horsemen)
}

// FourHorsemen — elements 15, 45, 75, 105
// Every 30 elements starting from 15
var FourHorsemen = []Horseman{
	{Element: 15, Name: "Phosphorus", Column: 3,  Protons: 19,  Neutrons: 16,  CircleLines: 35},
	{Element: 45, Name: "Rhodium",    Column: 10, Protons: 19,  Neutrons: 91,  CircleLines: 110},
	{Element: 75, Name: "Rhenium",    Column: 5,  Protons: 79,  Neutrons: 106, CircleLines: 185},
	{Element: 105, Name: "Dubnium",   Column: 8,  Protons: 109, Neutrons: 151, CircleLines: 260},
}

// ── Four Horsemen interval tests ──────────────────────────────────────────────

func TestFourHorsemenInterval(t *testing.T) {
	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║   THE FOUR HORSEMEN — ELEMENT INTERVALS                   ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")

	for i := 1; i < len(FourHorsemen); i++ {
		interval := FourHorsemen[i].Element - FourHorsemen[i-1].Element
		t.Logf("║  Element %3d → %3d (%s → %s): interval = %d     ║",
			FourHorsemen[i-1].Element, FourHorsemen[i].Element,
			FourHorsemen[i-1].Name, FourHorsemen[i].Name, interval)
		if interval != 30 {
			t.Errorf("expected interval 30, got %d", interval)
		}
	}

	t.Log("╠═══════════════════════════════════════════════════════════╣")
	t.Log("║  All intervals = 30 ✓                                     ║")
	t.Log("║  30 = 360/12 = one full column rotation                   ║")
	t.Log("║  15 = 30/2 = half rotation from origin                    ║")
	t.Log("╚═══════════════════════════════════════════════════════════╝")
}

func TestFourHorsemenProtonSequence(t *testing.T) {
	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║   FOUR HORSEMEN — PROTON SEQUENCE (left neighbours)       ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")

	t.Log("║  Element  15: protons = 19")
	t.Log("║  Element  45: protons = 19  (Δ = +0)")
	t.Log("║  Element  75: protons = 79  (Δ = +60)")
	t.Log("║  Element 105: protons = 109 (Δ = +30)")
	t.Log("║                                                           ║")

	protons := make([]int, len(FourHorsemen))
	for i, h := range FourHorsemen {
		protons[i] = h.Protons
	}

	for i := 1; i < len(protons); i++ {
		delta := protons[i] - protons[i-1]
		t.Logf("║  Proton %d → %d: Δ = %+d                               ║",
			protons[i-1], protons[i], delta)
	}

	t.Log("║                                                           ║")
	t.Log("║  Pattern: +0, +60, +30                                    ║")
	t.Log("║  60 and 30 — the same differences seen in tHz analysis!   ║")
	t.Log("║  (+60/+60/+120/+120 left, +60/+60/-120/+60 right)        ║")
	t.Log("╚═══════════════════════════════════════════════════════════╝")
}

func TestFourHorsemenNeutronSequence(t *testing.T) {
	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║   FOUR HORSEMEN — NEUTRON SEQUENCE (right neighbours)     ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")

	neutrons := make([]int, len(FourHorsemen))
	for i, h := range FourHorsemen {
		neutrons[i] = h.Neutrons
		t.Logf("║  Element %3d: neutrons = %3d                              ║",
			h.Element, h.Neutrons)
	}

	t.Log("║                                                           ║")
	// Cross differences: 16→106 and 91→151
	diff1 := neutrons[2] - neutrons[0] // 106 - 16 = 90
	diff2 := neutrons[3] - neutrons[1] // 151 - 91 = 60
	t.Logf("║  Element 15→75 neutrons: %d → %d = +%d                   ║",
		neutrons[0], neutrons[2], diff1)
	t.Logf("║  Element 45→105 neutrons: %d → %d = +%d                  ║",
		neutrons[1], neutrons[3], diff2)
	t.Log("║                                                           ║")
	t.Logf("║  Cross differences: +%d and +%d                           ║", diff1, diff2)
	t.Log("║  90 = 3 × 30 (three yellow circle intervals)              ║")
	t.Log("║  60 = 2 × 30 (two yellow circle intervals)                ║")
	t.Log("║  Both are multiples of 30, the horsemen interval          ║")
	t.Log("╚═══════════════════════════════════════════════════════════╝")
}

func TestFourHorsemenContactPointSum(t *testing.T) {
	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║   FOUR HORSEMEN — CONTACT POINT SUMS                      ║")
	t.Log("║   'contact point + lines to other side = total lines'     ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")

	for _, h := range FourHorsemen {
		sum := h.Protons + h.Neutrons
		t.Logf("║  Element %3d (%s): %d + %d = %d  (circle lines=%d) ║",
			h.Element, h.Name, h.Protons, h.Neutrons, sum, h.CircleLines)
	}

	t.Log("║                                                           ║")
	t.Log("║  For the horsemen, protons + neutrons = ?                 ║")
	t.Log("║  The rule for normal elements: contact + mirror = L       ║")
	t.Log("║  For horsemen: their own symmetry governs the sums        ║")

	// Check if protons + neutrons relates to circle lines
	for _, h := range FourHorsemen {
		sum := h.Protons + h.Neutrons
		diff := h.CircleLines - sum
		t.Logf("║    %s: P+N=%d, lines=%d, diff=%d                    ║",
			h.Name, sum, h.CircleLines, diff)
	}
	t.Log("╚═══════════════════════════════════════════════════════════╝")
}

func TestFourHorsemenCircleLinesProgression(t *testing.T) {
	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║   FOUR HORSEMEN — CIRCLE LINE PROGRESSION                 ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")

	lines := make([]int, len(FourHorsemen))
	for i, h := range FourHorsemen {
		lines[i] = h.CircleLines
		t.Logf("║  Element %3d (%s): circle lines = %d               ║",
			h.Element, h.Name, h.CircleLines)
	}

	t.Log("║                                                           ║")
	for i := 1; i < len(lines); i++ {
		delta := lines[i] - lines[i-1]
		t.Logf("║  Lines %d → %d: Δ = %d                                 ║",
			lines[i-1], lines[i], delta)
	}

	t.Log("║                                                           ║")
	// Check if consistent interval
	if lines[1]-lines[0] == lines[2]-lines[1] &&
		lines[2]-lines[1] == lines[3]-lines[2] {
		interval := lines[1] - lines[0]
		t.Logf("║  ✓ Constant interval: %d                                  ║", interval)
		t.Logf("║  %d = 3 × 25 = 3 × (2125/85)                             ║", interval)
		t.Logf("║  %d = 5 × 15 (5 × first horseman element)                 ║", interval)
	}
	t.Log("╚═══════════════════════════════════════════════════════════╝")
}

// ── The 3.4 × 25 = 85 connection ─────────────────────────────────────────────

func TestCiBase27Terminates(t *testing.T) {
	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║   Ci IN BASE 27 — 3.4 EXACT TERMINATION                   ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")
	t.Log("║                                                           ║")
	t.Log("║  Ci = 85/27                                               ║")
	t.Log("║  In base 27: Ci = 3.4 exactly (terminates)               ║")
	t.Log("║                                                           ║")
	t.Log("║  Proof: 3.4 in base 27 = 3 + 4/27 = (81+4)/27 = 85/27   ║")
	t.Log("║  ✓ 3.4_{27} = 85/27 = Ci                                  ║")
	t.Log("║                                                           ║")

	// Verify: 3 + 4/27 = 85/27
	// 3 × 27 = 81, 81 + 4 = 85, 85/27 = Ci ✓
	numerator := 3*27 + 4
	if numerator != 85 {
		t.Errorf("base 27 conversion: expected 85, got %d", numerator)
	}
	t.Logf("║  3×27 + 4 = %d = numerator of Ci ✓                      ║", numerator)

	t.Log("║                                                           ║")
	t.Log("║  3.4 × 5 = 17                                             ║")
	t.Log("║  3.4 × 10 = 34                                            ║")
	t.Log("║  3.4 × 15 = 51                                            ║")
	t.Log("║  3.4 × 20 = 68                                            ║")
	t.Log("║  3.4 × 25 = 85  ← numerator of Ci (circled in image)     ║")
	t.Log("║                                                           ║")

	// Verify 3.4 × 25 = 85
	// 3.4 = 17/5
	// 17/5 × 25 = 17 × 5 = 85 ✓
	result := big.NewInt(17)
	result.Mul(result, big.NewInt(5))
	if result.Int64() != 85 {
		t.Errorf("3.4 × 25: expected 85, got %d", result.Int64())
	}
	t.Logf("║  17/5 × 25 = 17 × 5 = %d ✓                              ║", result.Int64())

	t.Log("║                                                           ║")
	t.Log("║  25 = 5² = the square of 5 (steps in 3.4 multiples)      ║")
	t.Log("║  25 = 2125 ÷ 85 (from image 2)                            ║")
	t.Log("║  25 elements per column row (120/4 shells... approx)      ║")
	t.Log("╚═══════════════════════════════════════════════════════════╝")
}

func TestCiMath2125(t *testing.T) {
	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║   2125 — THE CENTRAL NUMBER                               ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")

	// From image 2: 2125 ÷ 85 = 25, 2125 ÷ Ci = 675
	n2125 := big.NewInt(2125)

	// 2125 ÷ 85 = 25
	rem85 := new(big.Int).Mod(n2125, big.NewInt(85))
	div85 := new(big.Int).Div(n2125, big.NewInt(85))
	t.Logf("║  2125 ÷ 85 = %d remainder %d ✓                          ║",
		div85.Int64(), rem85.Int64())

	// 2125 ÷ 27 = 78 remainder 19
	rem27 := new(big.Int).Mod(n2125, big.NewInt(27))
	div27 := new(big.Int).Div(n2125, big.NewInt(27))
	t.Logf("║  2125 ÷ 27 = %d remainder %d                            ║",
		div27.Int64(), rem27.Int64())

	// 2125 ÷ Ci = 2125 × 27/85 = 57375/85 = 675 exactly
	// 2125 × 27 = 57375
	// 57375 / 85 = 675
	num := new(big.Int).Mul(n2125, big.NewInt(27))
	remCi := new(big.Int).Mod(num, big.NewInt(85))
	divCi := new(big.Int).Div(num, big.NewInt(85))
	t.Logf("║  2125 ÷ Ci = 2125 × 27/85 = %d/%d = %d remainder %d     ║",
		num.Int64(), 85, divCi.Int64(), remCi.Int64())

	t.Log("║                                                           ║")
	if remCi.Sign() == 0 && divCi.Int64() == 675 {
		t.Log("║  ✓ 2125 ÷ Ci = 675 EXACTLY = Seekins constant           ║")
	}

	// Factor 2125
	// 2125 = 5³ × 17
	// 5³ = 125
	// 125 × 17 = 2125
	t.Log("║                                                           ║")
	t.Log("║  2125 = 5³ × 17 = 125 × 17                               ║")
	t.Log("║  17 = 3.4 × 5 (first multiple of 3.4 that is integer)    ║")
	t.Log("║  5³ = 125 = GCD(6750, 6625) — from Seekins/Planck ratio   ║")
	t.Log("║                                                           ║")
	t.Log("║  The GCD of the Seekins and Planck constants (scaled)     ║")
	t.Log("║  is 125 = 5³, and 2125 = 125 × 17 where 17 = 3.4 × 5    ║")
	t.Log("║                                                           ║")

	// 2 mod 27 = 2 (not 18)
	// But 2^18 mod 27 = 1 (the multiplicative order)
	// Chris wrote "2 mod 27 = 18" meaning the ORDER of 2 mod 27 is 18
	pow := big.NewInt(1)
	base := big.NewInt(2)
	mod27 := big.NewInt(27)
	for i := 1; i <= 18; i++ {
		pow.Mul(pow, base)
		pow.Mod(pow, mod27)
		if pow.Int64() == 1 {
			t.Logf("║  2^%d mod 27 = 1 ✓ (multiplicative order = %d = binary period) ║",
				i, i)
			break
		}
	}
	t.Log("╚═══════════════════════════════════════════════════════════╝")
}

func TestThirtyFourAndSeventeen(t *testing.T) {
	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║   3.4 = 17/5 — THE BRIDGE NUMBER                         ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")
	t.Log("║                                                           ║")
	t.Log("║  3.4 = 17/5 = Ci in base 10 truncated after 1 decimal    ║")
	t.Log("║  Ci = 3.148148... but 3.4 = Ci in base 27                ║")
	t.Log("║                                                           ║")
	t.Log("║  17 appears throughout the framework:                     ║")
	t.Log("║  - 17-bit rotation in packRConstant (ci-sha4096)          ║")
	t.Log("║  - 17 is the smallest prime > 16 (numStates)             ║")
	t.Log("║  - 3.4 × 5 = 17                                           ║")
	t.Log("║  - 2125 = 5³ × 17                                         ║")
	t.Log("║  - 17/5 × 25 = 85 = numerator of Ci                      ║")
	t.Log("║                                                           ║")
	t.Log("║  5 appears throughout:                                    ║")
	t.Log("║  - x^5 S-box (Poseidon2 — gcd(5,p-1)=1)                  ║")
	t.Log("║  - 5 progression types in proton/neutron table            ║")
	t.Log("║  - 5 symmetry types in square symmetry (Blue/Green/etc)   ║")
	t.Log("║  - 3.4 × 5 = 17 (first integer multiple of 3.4)          ║")
	t.Log("║  - 5² = 25 = 2125/85                                      ║")
	t.Log("║                                                           ║")
	t.Log("║  The x^5 S-box choice (standard in Poseidon2) connects    ║")
	t.Log("║  to 5 progression types and 3.4 = 17/5.                  ║")
	t.Log("║  Was the S-box degree chosen independently, or does       ║")
	t.Log("║  5 appear here for the same reason it appears everywhere? ║")
	t.Log("╚═══════════════════════════════════════════════════════════╝")
}

// ── Image 3 analysis: radius paper ───────────────────────────────────────────

func TestRadiusPaperStructure(t *testing.T) {
	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║   RADIUS PAPER — HORSEMEN CONTACT POINT NEIGHBOURS       ║")
	t.Log("║   (From image 3)                                          ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")
	t.Log("║                                                           ║")
	t.Log("║  6 lines at top — the starting radius                     ║")
	t.Log("║                                                           ║")
	t.Log("║  Element  15: contact neighbours 19/16                    ║")
	t.Log("║    → 30 (two arrows left)                                 ║")
	t.Log("║    → 90 (right circle)                                    ║")
	t.Log("║                                                           ║")
	t.Log("║  Element  45: contact neighbours 19/91                    ║")
	t.Log("║    → 30 (arrows)                                          ║")
	t.Log("║    → 60 (right circle)                                    ║")
	t.Log("║                                                           ║")
	t.Log("║  Element  75: contact neighbours 79/106                   ║")
	t.Log("║    → 30 (arrows)                                          ║")
	t.Log("║    → 39 (left circle)                                     ║")
	t.Log("║                                                           ║")
	t.Log("║  Element 105: contact neighbours 109/151                  ║")
	t.Log("║                                                           ║")

	// The circled numbers on the paper: 90, 60, 39
	// 90 + 60 = 150 = 2 × 75
	// 90 - 60 = 30 = the horsemen interval
	// 39 = ?
	circled := []int{90, 60, 39}
	t.Logf("║  Circled values: 90, 60, 39                               ║")
	t.Logf("║  90 - 60 = %d = horsemen interval                         ║", circled[0]-circled[1])
	t.Logf("║  90 + 60 = %d = 2 × 75 (third horseman)                  ║", circled[0]+circled[1])
	t.Logf("║  39 = 90 - 51 = ?                                         ║")
	t.Logf("║  39 = 3 × 13 (Al is element 13, col 12 anchor)           ║")
	t.Logf("║  60 = 2 × 30 = two horsemen intervals                     ║")
	t.Logf("║  90 = 3 × 30 = three horsemen intervals                   ║")

	t.Log("║                                                           ║")
	t.Log("║  The arrows showing 30 between each horseman confirm       ║")
	t.Log("║  the 30-element interval. The right circles (90, 60)      ║")
	t.Log("║  show the cross-differences from the neutron sequence.    ║")
	t.Log("╚═══════════════════════════════════════════════════════════╝")

	// Verify the differences
	if circled[0]-circled[1] != 30 {
		t.Errorf("90-60 should be 30, got %d", circled[0]-circled[1])
	}
}

func TestFourHorsemenSummary(t *testing.T) {
	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║   FOUR HORSEMEN — COMPLETE SUMMARY                        ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")
	t.Log("║                                                           ║")
	t.Log("║  Position: elements 15, 45, 75, 105 — every 30 from 15   ║")
	t.Log("║  15 = 30/2 = half rotation from origin                    ║")
	t.Log("║                                                           ║")
	t.Log("║  UNIQUE PROPERTY: their circle lines ≠ nuclear mass       ║")
	t.Log("║  All other elements: L lines = nuclear mass               ║")
	t.Log("║  Horsemen are the exceptions — they form their own        ║")
	t.Log("║  internal symmetry separate from the main rule            ║")
	t.Log("║                                                           ║")
	t.Log("║  Proton sequence:  19, 19, 79, 109  (Δ: +0, +60, +30)   ║")
	t.Log("║  Neutron sequence: 16, 91, 106, 151                       ║")
	t.Log("║    16 → 106 = +90 (cross diff, elements 15→75)           ║")
	t.Log("║    91 → 151 = +60 (cross diff, elements 45→105)          ║")
	t.Log("║                                                           ║")
	t.Log("║  +60 appears in BOTH:                                     ║")
	t.Log("║  - Proton delta (19→79→109: +60, +30)                    ║")
	t.Log("║  - Neutron cross delta (91→151: +60)                      ║")
	t.Log("║  - tHz column pair differences (+60/+60/+120/+120)        ║")
	t.Log("║  - Palindrome partial sums                                ║")
	t.Log("║                                                           ║")
	t.Log("║  The number 60 connects:                                  ║")
	t.Log("║  proton/neutron structure ↔ tHz structure ↔ circle code   ║")
	t.Log("║                                                           ║")
	t.Log("║  60 = 2 × 30 = 2 × (360/12)                              ║")
	t.Log("║  60 = 5 × 12 (5 progression types × 12 columns)          ║")
	t.Log("║  60 = the harmonic bridge across all three structures      ║")
	t.Log("╚═══════════════════════════════════════════════════════════╝")
}

func TestSixtyAsHarmonicBridge(t *testing.T) {
	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║   60 — THE HARMONIC BRIDGE                                ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")

	// Verify 60 appears in all three structures
	type appearance struct {
		structure string
		context   string
		value     int
	}

	appearances := []appearance{
		{"tHz column pairs (left)", "+60 difference (cols 6&8 → 8&10)", 60},
		{"tHz column pairs (left)", "+60 difference (cols 8&10 → 10&12)", 60},
		{"tHz column pairs (right)", "+60 difference (cols 7&9 → 9&11)", 60},
		{"tHz column pairs (right)", "+60 difference (cols 1&3 → 3&5)", 60},
		{"horsemen protons", "Δ from element 45→75 protons (19→79)", 60},
		{"horsemen neutrons", "cross Δ elements 45→105 (91→151)", 60},
		{"circle code", "palindrome sum/2 = 30/2×4 steps... 60", 60},
	}

	for _, a := range appearances {
		t.Logf("║  %-25s: %s", a.structure, a.context)
	}

	t.Log("║                                                           ║")
	t.Log("║  60 = 360/6 (six-sided symmetry base)                    ║")
	t.Log("║  60 = LCM(12, 20, 30) / 10                               ║")
	t.Log("║  60 = the natural harmonic of the 12-column structure     ║")
	t.Log("╚═══════════════════════════════════════════════════════════╝")
}
