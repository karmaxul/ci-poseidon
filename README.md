# ci-Poseidon

**Rational Constant Structures for Arithmetization-Oriented Hash Functions**

A Poseidon2-style ZK-friendly hash function using constants derived deterministically
from the rational approximation Ci = 85/27, benchmarked against Poseidon2 on BN254,
BLS12-381, and Goldilocks fields.

Paper: [IACR ePrint 2026/109784](https://eprint.iacr.org/2026/109784) (pending approval)
Repository: https://github.com/karmaxul/ci-poseidon

---

## Key Results

| Metric | Result |
|---|---|
| Avalanche (BN254) | 50.02% — statistically ideal |
| Avalanche (BLS12-381) | 49.93% |
| Avalanche (Goldilocks) | 50.28% |
| R1CS constraint reduction vs Poseidon2 (t=3) | 18.8% |
| R1CS constraint reduction vs Poseidon2 (t=4) | 25.8% |
| R1CS constraint reduction vs Poseidon2 (t=6) | 28.8% |
| Groth16 prover time (all widths) | 2.48–2.66 ms (flat across widths) |
| Groth16 verify time | < 560 µs |
| Proof size | ~127 bytes (constant) |
| Collisions (500 inputs) | 0 |

Constraint counts verified via `gnark v0.15.0 GetNbConstraints()`.

---

## Constant Generation

Constants are derived from Ci = 85/27 reduced modulo the target field prime:

```
K[i] = floor(85 · pᵢ · 2⁶⁴) · (27(pᵢ + 1))⁻¹  mod p
```

where `pᵢ` is the i-th prime. No floating-point arithmetic. No LFSR seed trust required.
Every constant is independently verifiable from first principles.

**Example — BN254, K[0]:**
```
K[0] = (85 · 2 · 2⁶⁴) · (27 · 3)⁻¹  mod p
     = 0x0eef8d269dff839b19482eccdf4786bac6e09d069106aaa7084f38d52a781949
```

---

## Quickstart

**Requirements:** Go 1.21+, gnark v0.15.0

```bash
git clone https://github.com/karmaxul/ci-poseidon
cd ci-poseidon

# Run full test suite (avalanche, collision, R1CS, field arithmetic)
go test ./... -v

# Run Groth16 benchmarks
go test ./gnark/... -bench=. -benchtime=10s

# Generate Circom circuits for all widths
go run circom_export.go
```

---

## Supported Fields

| Field | Prime | Used by |
|---|---|---|
| BN254 | 254-bit | gnark, circom, Ethereum precompiles |
| BLS12-381 | 255-bit | Ethereum PoS, Zcash, Filecoin |
| Goldilocks | 2⁶⁴ − 2³² + 1 | Plonky2, Plonky3, STARK systems |

---

## Variable-Width Sponge

ci-poseidon introduces a variable-width sponge that adapts state geometry based on
measured diffusion. The sponge starts at t=2 and expands through t ∈ {2, 3, 4, 6}.

Prover time is flat across all widths — expansion from t=2 to t=6 carries near-zero
ZK overhead. Observed width history across a 20-input stream:

```
[2, 3, 4, 6, 6, 6, 6, 6, ...]
```

The sponge stabilises at t=6 within 4 absorb operations.

---

## Generated Circom Circuits (BN254)

| Circuit | Width | Constants | Constraints |
|---|---|---|---|
| ci_poseidon_t2_bn254.circom | 2 | 130 | 218 |
| ci_poseidon_t3_bn254.circom | 3 | 147 | 195 |
| ci_poseidon_t4_bn254.circom | 4 | 164 | 196 |
| ci_poseidon_t6_bn254.circom | 6 | 198 | 222 |

---

## Deployed Verifier Contracts (Ethereum Sepolia)

On-chain Groth16 verifiers deployed and verified for all four widths:

| Width | Contract Address | Status |
|---|---|---|
| t=2 | `0x43bBb210d78A5Ce79F99741D6335B478194080D0` | ✅ Verified |
| t=3 | `0x75cc3fF2905e328E199Fda66c200244112147084` | ✅ Verified |
| t=4 | `0x310273499087dDd432156013478d6B2c7ac15567` | ✅ Verified |
| t=6 | `0x77A05B1d95a91607048700715438932eDb772ced` | ✅ Verified |

```solidity
verifyProof(uint256[2], uint256[2][2], uint256[2], uint256[1]) returns (bool)
```

---

## Repository Contents

```
ci_poseidon.go          — Core construction (K-layer, R-layer, permutation, sponge)
ci_poseidon_test.go     — 87+ tests (avalanche, collision, R1CS, field arithmetic)
circom_export.go        — Circom 2.1.0 circuit generator
circuits/               — Generated Circom circuits (BN254, all widths)
gnark/                  — gnark Groth16 circuit and benchmarks
RESEARCH/               — MDS derivation, symmetry analysis, field arithmetic notes
```

---

## MDS Matrix (Section 7)

The MDS matrices at all supported widths are derived from a 19-element subset whose
spectral properties produce full MDS rank without augmentation across BN254, BLS12-381,
and Goldilocks simultaneously. Full derivation and verification scripts are in `RESEARCH/`.

---

## License

MIT — Christopher Seekins, 2026.

## Related

- [ci-sha4096](https://github.com/karmaxul/ci-sha4096) — companion 4096-bit classical hash, IACR ePrint 2026/109810
