# ci-Poseidon: Rational Constant Structures for Arithmetization-Oriented Hash Functions

**GitHub:** https://github.com/karmaxul/ci-poseidon  
**Paper:** [ci-Poseidon ePrint](https://eprint.iacr.org/2026/XXXX) (latest revision)

### Project Family

This repository is part of the **HealChain / Harmony Worldwide Cryptographic Suite**:

- **[HealChain-crypto](https://github.com/karmaxul/HealChain-crypto)** — Umbrella repo — cryptographic suite overview
- **[ci-sha4096](https://github.com/karmaxul/ci-sha4096)** — 4096-bit classical hash, IACR ePrint 2026/109810
- **[ci-poseidon](https://github.com/karmaxul/ci-poseidon)** — Arithmetization-oriented ZK-friendly hash, IACR ePrint (pending)
- **[ci-plonky3](https://github.com/karmaxul/ci-plonky3)** — STARK variant over Goldilocks/Plonky3 — K-sequence 5.1% faster than Grain LFSR at t=12
- **[ci-quantum-storage](https://github.com/karmaxul/ci-quantum-storage)** — Classical stabilizer / reference implementation
- **[HealChain](https://healchain.org)** — Production deployment using ci-poseidon for on-chain ZK commitments

All projects derive constants from the same **Harmony Worldwide mathematical framework** (Ci = 85/27 + resonance matrix).

# ci-poseidon

ZK-friendly hash construction using rational constants derived from Ci = 85/27 — arithmetization-oriented variant of ci-sha4096.

IACR ePrint: *submission pending*
Related: [ci-sha4096](https://github.com/karmaxul/ci-sha4096) — IACR ePrint 2026/109810

---

## Overview

ci-poseidon applies the [Harmony Worldwide](https://healchain.org/force/quantum-computing) rational constant framework to a Poseidon2-style arithmetization-oriented hash function. It is designed to compose with ci-sha4096 in the HealChain architecture:

```
ci-sha4096  →  4096-bit off-chain integrity digest (recovery blueprint)
ci-poseidon →  ZK-provable field element commitment (on-chain verification)
```

---

## Key Results

| Metric | Result |
|---|---|
| Avalanche (BN254) | 50.02% — statistically ideal |
| Avalanche (BLS12-381) | 49.93% |
| Avalanche (Goldilocks) | 50.28% |
| R1CS savings vs Poseidon2 (t=3) | 18.8% |
| R1CS savings vs Poseidon2 (t=4) | 25.8% |
| R1CS savings vs Poseidon2 (t=6) | 28.8% |
| Groth16 prover time (all widths) | 2.48–2.66 ms (flat) |
| Groth16 verify time | < 560 µs |
| Proof size | ~127 bytes (constant) |
| Collisions (500 inputs) | 0 |

Constraint counts compiler-verified via gnark v0.15.0 `GetNbConstraints()`.

---

## Constant Generation

All round constants are derived from Ci = 85/27 reduced modulo the target field prime:

```
K[i] = floor(85 · pᵢ · 2⁶⁴) · (27(pᵢ + 1))⁻¹ mod p
```

where pᵢ is the i-th prime. No floating-point arithmetic is involved. No LFSR seed trust is required. Any constant is independently verifiable from first principles.

**Example (BN254, K[0]):**
```
K[0] = (85 · 2 · 2⁶⁴) · (27 · 3)⁻¹ mod p
     = 0x0eef8d269dff839b19482eccdf4786bac6e09d069106aaa7084f38d52a781949
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

ci-poseidon introduces a variable-width sponge that adapts state geometry based on measured diffusion. The sponge starts at t=2 and expands through t ∈ {2, 3, 4, 6} using thresholds derived from the Harmony Worldwide resonance matrix.

Observed width history across a 20-input stream:
```
[2, 3, 4, 6, 6, 6, 6, 6, ...]
```

The sponge stabilises at t=6 within 4 absorb operations. Prover time is flat across all widths — expansion from t=2 to t=6 carries near-zero ZK cost.

---

## Circuit Generation

Production Circom 2.1.0 circuits for all four widths are generated automatically by `circom_export.go`:

| Circuit | Width | Constants | Constraints |
|---|---|---|---|
| ci_poseidon_t2_bn254.circom | 2 | 130 | 218 |
| ci_poseidon_t3_bn254.circom | 3 | 147 | 195 |
| ci_poseidon_t4_bn254.circom | 4 | 164 | 196 |
| ci_poseidon_t6_bn254.circom | 6 | 198 | 222 |

---

## Square Symmetry MDS

Section 7 of the paper presents a standalone result: the tHz wavelengths of the 19 elements whose class position is shared across all three square groups of the Harmony Worldwide resonance matrix produce a **natural MDS matrix at every supported width (t ∈ {2, 3, 4, 6}) without augmentation**.

The 12-column subclass sequence `2,2,4,3,1,3,3,1,3,4,2,2` is a perfect palindrome centred on the Nitrogen column. The average tHz of the 19 shared elements gravitates to within 7.0 units of Oxygen — the sponge anchor for t=3 and the element closest to the midpoint of the full resonance matrix range.

These properties were not designed. They were found.

---

## Repository Contents

```
ci_poseidon.go          — Core construction (K-layer, R-layer, permutation, sponge)
ci_poseidon_test.go     — 87+ tests (avalanche, collision, R1CS, field arithmetic)
circom_export.go        — Circom 2.1.0 circuit generator
circuits/               — Generated Circom circuits (BN254, all widths)
gnark/                  — gnark Groth16 circuit and benchmark
RESEARCH/               — Square symmetry MDS derivation and analysis
```

---

## Deployed Verifier Contracts (Sepolia)

| Width | Contract Address | Status |
|---|---|---|
| t=2 | `0x43bBb210d78A5Ce79F99741D6335B478194080D0` | ✅ Verified |
| t=3 | `0x75cc3fF2905e328E199Fda66c200244112147084` | ✅ Verified |
| t=4 | `0x310273499087dDd432156013478d6B2c7ac15567` | ✅ Verified |
| t=6 | `0x77A05B1d95a91607048700715438932eDb772ced` | ✅ Verified |

Call `verifyProof(uint256[2],uint256[2][2],uint256[2],uint256[1])` returns `bool`.
Deployed June 2026 on Ethereum Sepolia testnet.

---

## Related Work

- **ci-sha4096** — 4096-bit off-chain integrity hash, IACR ePrint 2026/109810
  https://github.com/karmaxul/ci-sha4096
- **HealChain** — Distributed self-healing storage using both constructions
  https://healchain.org
- **Harmony Worldwide** — The mathematical framework underlying both constructions
  https://healchain.org/force/quantum-computing

---

## License

MIT — Christopher Seekins, Harmony Worldwide, 2026.

> "The symmetry is not in the presentation. It is in the structure of matter itself."
