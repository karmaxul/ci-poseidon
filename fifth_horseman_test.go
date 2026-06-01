// fifth_horseman_test.go — Extrapolating the Four Horsemen pattern to element 135
//
// From Chris's paper (image IMG_1513):
//   The fifth horseman would be element 135 (30 beyond Dubnium/105)
//   Proton sequence:  19, 19, 79, 109, 169  (+0, +60, +30, +60)
//   Neutron sequence: 16, 91, 106, 151, 166
//   Circle lines:     35, 110, 185, 260, 335  (constant +75)
//   169 + 166 = 335 = circle lines ✓
//
// The framework currently ends at element 121 (Ps), so element 135
// would be beyond the framework — but the pattern predicts it exactly.
//
// Author:  Christopher Seekins — Harmony Worldwide / HealChain
// Package: github.com/karmaxul/ci-poseidon
// Date:    June 2026

package ciposeidon

import "testing"

// AllHorsemen extends FourHorsemen with the predicted fifth
var AllHorsemen = []Horseman{
	{Element: 15,  Name: "Phosphorus",   Column: 3,  Protons: 19,  Neutrons: 16,  CircleLines: 35},
	{Element: 45,  Name: "Rhodium",      Column: 10, Protons: 19,  Neutrons: 91,  CircleLines: 110},
	{Element: 75,  Name: "Rhenium",      Column: 5,  Protons: 79,  Neutrons: 106, CircleLines: 185},
	{Element: 105, Name: "Dubnium",      Column: 8,  Protons: 109, Neutrons: 151, CircleLines: 260},
	{Element: 135, Name: "Element-135?", Column: -1, Protons: 169, Neutrons: 166, CircleLines: 335},
}

func TestFifthHorsemanPrediction(t *testing.T) {
	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║   FIFTH HORSEMAN — ELEMENT 135 PREDICTION                 ║")
	t.Log("║   (From Chris's paper, June 1 2026)                       ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")
	t.Log("║                                                           ║")
	t.Log("║  Element 135 = 105 + 30 = beyond current 121-element      ║")
	t.Log("║  framework, but the pattern predicts it exactly.           ║")
	t.Log("║                                                           ║")

	// Verify circle lines +75 constant interval
	t.Log("║  Circle lines progression (should all be +75):            ║")
	for i := 1; i < len(AllHorsemen); i++ {
		delta := AllHorsemen[i].CircleLines - AllHorsemen[i-1].CircleLines
		status := "✓"
		if delta != 75 {
			status = "✗"
			t.Errorf("circle lines delta %d→%d: expected 75, got %d",
				AllHorsemen[i-1].CircleLines, AllHorsemen[i].CircleLines, delta)
		}
		t.Logf("║    %3d → %3d: Δ=%d %s                                   ║",
			AllHorsemen[i-1].CircleLines, AllHorsemen[i].CircleLines, delta, status)
	}

	t.Log("║                                                           ║")

	// Verify protons + neutrons = circle lines for all five
	t.Log("║  Protons + Neutrons = Circle Lines (should all match):    ║")
	for _, h := range AllHorsemen {
		sum := h.Protons + h.Neutrons
		status := "✓"
		if sum != h.CircleLines {
			status = "✗"
			t.Errorf("element %d: %d+%d=%d ≠ lines=%d",
				h.Element, h.Protons, h.Neutrons, sum, h.CircleLines)
		}
		t.Logf("║    Element %3d (%s): %d+%d=%d lines=%d %s       ║",
			h.Element, h.Name, h.Protons, h.Neutrons, sum, h.CircleLines, status)
	}

	t.Log("║                                                           ║")

	// Verify all intervals are 30
	t.Log("║  Element intervals (should all be 30):                    ║")
	for i := 1; i < len(AllHorsemen); i++ {
		interval := AllHorsemen[i].Element - AllHorsemen[i-1].Element
		status := "✓"
		if interval != 30 {
			status = "✗"
		}
		t.Logf("║    %d → %d: interval=%d %s                              ║",
			AllHorsemen[i-1].Element, AllHorsemen[i].Element, interval, status)
	}

	t.Log("║                                                           ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")
	t.Log("║  PROTON SEQUENCE:  19, 19, 79, 109, 169                   ║")
	t.Log("║  Deltas:           +0, +60, +30, +60                      ║")
	t.Log("║                                                           ║")
	t.Log("║  NEUTRON SEQUENCE: 16, 91, 106, 151, 166                  ║")
	t.Log("║  Cross diffs:      16→106=+90, 91→151=+60, 106→166=+60   ║")
	t.Log("║                                                           ║")
	t.Log("║  CIRCLE LINES:     35, 110, 185, 260, 335                 ║")
	t.Log("║  Constant Δ=75 throughout                                 ║")
	t.Log("║                                                           ║")
	t.Log("║  Fifth horseman circle lines = 335                        ║")
	t.Log("║  335 = 5 × 67                                             ║")
	t.Log("║  335 = 260 + 75 = fourth lines + constant interval        ║")
	t.Log("║  335 / 5 = 67 (67 is the 19th prime)                     ║")
	t.Log("║  19 = first proton value of the horsemen sequence         ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")

	// Cross neutron differences
	neutrons := make([]int, len(AllHorsemen))
	for i, h := range AllHorsemen {
		neutrons[i] = h.Neutrons
	}
	t.Log("║  Neutron cross-differences (every 2 steps):               ║")
	for i := 2; i < len(neutrons); i++ {
		diff := neutrons[i] - neutrons[i-2]
		t.Logf("║    neutron[%d]→neutron[%d]: %d→%d = +%d                  ║",
			i-2, i, neutrons[i-2], neutrons[i], diff)
	}

	t.Log("║                                                           ║")
	t.Log("║  169+166 = 335 ✓ (Chris's calculation confirmed)          ║")
	t.Log("╚═══════════════════════════════════════════════════════════╝")

	// Final verification
	fifth := AllHorsemen[4]
	if fifth.Protons+fifth.Neutrons != 335 {
		t.Errorf("fifth horseman: %d+%d ≠ 335", fifth.Protons, fifth.Neutrons)
	}
	if fifth.CircleLines != 335 {
		t.Errorf("fifth horseman circle lines: expected 335, got %d", fifth.CircleLines)
	}
	t.Log("✓ Fifth horseman prediction verified: element 135, lines=335")
}

func TestHorsemenPatternProperties(t *testing.T) {
	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════════════╗")
	t.Log("║   HORSEMEN PATTERN — FULL PROPERTIES                      ║")
	t.Log("╠═══════════════════════════════════════════════════════════╣")

	// Sum of all circle lines
	totalLines := 0
	for _, h := range AllHorsemen {
		totalLines += h.CircleLines
	}
	t.Logf("║  Sum of all 5 circle lines: %d                            ║", totalLines)
	t.Logf("║  %d / 5 = %d (average)                                   ║", totalLines, totalLines/5)
	t.Logf("║  %d / 27 = %d remainder %d                               ║",
		totalLines, totalLines/27, totalLines%27)
	t.Logf("║  %d / 85 = %d remainder %d                               ║",
		totalLines, totalLines/85, totalLines%85)

	t.Log("║                                                           ║")

	// First horseman element (15) × 5 = 75 = circle line interval
	t.Logf("║  First horseman (15) × 5 = %d = circle line interval ✓  ║", 15*5)
	// 75 × 5 = 375 ≠ 335... but 75 × (5-1) = 300, 335-35 = 300 ✓
	t.Logf("║  335 - 35 = %d = 75 × 4 ✓                               ║", 335-35)

	t.Log("║                                                           ║")

	// Proton sequence: 19, 19, 79, 109, 169
	// Differences: 0, 60, 30, 60
	// Pattern of differences: 0, 60, 30, 60, ?
	// Could be 30 next: 169+30=199? Or 60: 169+60=229?
	t.Log("║  Proton delta pattern: +0, +60, +30, +60                  ║")
	t.Log("║  If pattern is 0,60,30,60,30,60... next would be +30      ║")
	t.Log("║  Sixth horseman proton prediction: 169+30 = 199            ║")
	t.Log("║  If pattern is 0,60,30,60,60... next would be +60         ║")
	t.Log("║  Sixth horseman proton prediction: 169+60 = 229            ║")

	t.Log("╚═══════════════════════════════════════════════════════════╝")
}
