//! ci-poseidon STARK variant — Plonky3/Goldilocks implementation
//!
//! Two constant generation methods are benchmarked against Plonky3's default
//! Grain LFSR constants:
//!
//!   1. K-sequence (Ci=85/27): verifiable from first principles via prime sequence
//!   2. Periodic table (tHz): column anchor wavelengths from the Harmony Worldwide
//!      resonance matrix — pure physical constants, no mathematical derivation needed
//!
//! tHz constants for t=8 use columns 3-10 (Be→Na):
//!   Pairs sum to 1107 tHz — perfect mirror symmetry across the center.
//!
//! tHz constants for t=12 use all 12 columns (He→Al):
//!   Every pair also sums to 1107 tHz — same balance constant at both widths.
//!
//! Field: Goldilocks (p = 2^64 - 2^32 + 1 = 18446744069414584321)
//! S-box: x^7   rf=8   rp=22 (Plonky3 standard for Goldilocks)
//!
//! Author:  Christopher Seekins — Harmony Worldwide / HealChain
//! Date:    June 2026

use p3_field::{Field, PrimeCharacteristicRing, PrimeField64};
use p3_goldilocks::Goldilocks;
use p3_goldilocks::{
    Poseidon2Goldilocks,
    GOLDILOCKS_POSEIDON2_HALF_FULL_ROUNDS,
    GOLDILOCKS_POSEIDON2_PARTIAL_ROUNDS_8,
    GOLDILOCKS_POSEIDON2_PARTIAL_ROUNDS_12,
    default_goldilocks_poseidon2_8,
    default_goldilocks_poseidon2_12,
};
use p3_poseidon2::{ExternalLayerConstants, Poseidon2};
use p3_symmetric::Permutation;

// ── Goldilocks field prime ────────────────────────────────────────────────────

const GOLDILOCKS_PRIME: u64 = 18_446_744_069_414_584_321;

// ── tHz anchor values (×10 for exact integer representation) ─────────────────

/// Column anchor tHz values × 10 for t=8 (columns 3-10: Be→Na)
/// Pairs sum to 11070 (= 1107.0 × 10) — perfect mirror symmetry.
/// Be(3)↔Na(10): 6585+4485=11070
/// B(4)↔Ne(9):   6285+4785=11070
/// C(5)↔F(8):    5985+5085=11070
/// N(6)↔O(7):    5685+5385=11070
const THZ_T8: [u64; 8] = [
    6585, // Be  col 3
    6285, // B   col 4
    5985, // C   col 5
    5685, // N   col 6
    5385, // O   col 7
    5085, // F   col 8
    4785, // Ne  col 9
    4485, // Na  col 10
];

/// Column anchor tHz values × 10 for t=12 (columns 1-12: He→Al)
/// Every pair sums to 11070 — same balance constant as t=8.
/// He(1)↔Al(12): 7185+3885=11070
/// Li(2)↔Mg(11): 6885+4185=11070
/// Be(3)↔Al(10): 6585+4485=11070  (same inner 8 as t=8)
/// ...all 6 pairs sum to 11070
const THZ_T12: [u64; 12] = [
    7185, // He  col 1
    6885, // Li  col 2
    6585, // Be  col 3
    6285, // B   col 4
    5985, // C   col 5
    5685, // N   col 6
    5385, // O   col 7
    5085, // F   col 8
    4785, // Ne  col 9
    4485, // Na  col 10
    4185, // Mg  col 11
    3885, // Al  col 12
];

// ── Goldilocks field helpers ──────────────────────────────────────────────────

fn mod_pow(mut base: u128, mut exp: u128, modulus: u128) -> u128 {
    let mut result = 1u128;
    base %= modulus;
    while exp > 0 {
        if exp & 1 == 1 { result = result * base % modulus; }
        exp >>= 1;
        base = base * base % modulus;
    }
    result
}

fn first_primes(n: usize) -> Vec<u64> {
    let n = n.min(512);
    let mut primes = Vec::with_capacity(n);
    let mut sieve = vec![true; 3700];
    sieve[0] = false; sieve[1] = false;
    for i in 2..sieve.len() {
        if sieve[i] {
            primes.push(i as u64);
            if primes.len() == n { break; }
            let mut j = i * i;
            while j < sieve.len() { sieve[j] = false; j += i; }
        }
    }
    primes
}

// ── K-constant generation (Ci=85/27) ─────────────────────────────────────────

/// K[i] = (85 * prime[i] * 2^64) * inv(27*(prime[i]+1)) mod p
/// 2^64 mod p = 2^32 - 1 = 4294967295 (since p = 2^64 - 2^32 + 1)
pub fn k_constant(prime: u64) -> Goldilocks {
    let p = GOLDILOCKS_PRIME as u128;
    let pow64_mod_p: u128 = 4_294_967_295;
    let num = (85u128 * prime as u128 % p) * pow64_mod_p % p;
    let denom = (27u128 * (prime as u128 + 1)) % p;
    let denom_inv = mod_pow(denom, p - 2, p);
    Goldilocks::new((num * denom_inv % p) as u64)
}

pub fn generate_k_constants(count: usize) -> Vec<Goldilocks> {
    first_primes(count).iter().map(|&p| k_constant(p)).collect()
}

// ── tHz constant generation (periodic table) ──────────────────────────────────

/// Convert a tHz×10 integer to a Goldilocks field element.
/// The value is already a small integer so no modular arithmetic needed.
fn thz_constant(thz10: u64) -> Goldilocks {
    Goldilocks::new(thz10)
}

// ── External layer constants ──────────────────────────────────────────────────

/// Build external constants from a flat slice of Goldilocks values.
fn make_external<const WIDTH: usize>(
    values: &[Goldilocks],
) -> ExternalLayerConstants<Goldilocks, WIDTH> {
    let half_rf = GOLDILOCKS_POSEIDON2_HALF_FULL_ROUNDS; // 4
    assert!(values.len() >= 2 * half_rf * WIDTH);

    let mut initial: Vec<[Goldilocks; WIDTH]> = Vec::with_capacity(half_rf);
    let mut terminal: Vec<[Goldilocks; WIDTH]> = Vec::with_capacity(half_rf);

    for r in 0..half_rf {
        let mut arr = [Goldilocks::ZERO; WIDTH];
        for i in 0..WIDTH { arr[i] = values[r * WIDTH + i]; }
        initial.push(arr);
    }
    for r in 0..half_rf {
        let mut arr = [Goldilocks::ZERO; WIDTH];
        for i in 0..WIDTH { arr[i] = values[(half_rf + r) * WIDTH + i]; }
        terminal.push(arr);
    }
    ExternalLayerConstants::new(initial, terminal)
}

// ── K-sequence permutation constructors ──────────────────────────────────────

fn k_external_constants_8() -> ExternalLayerConstants<Goldilocks, 8> {
    let total = 2 * GOLDILOCKS_POSEIDON2_HALF_FULL_ROUNDS * 8; // 64
    let vals: Vec<Goldilocks> = generate_k_constants(total);
    make_external::<8>(&vals)
}

fn k_internal_constants_8() -> Vec<Goldilocks> {
    let offset = 2 * GOLDILOCKS_POSEIDON2_HALF_FULL_ROUNDS * 8; // 64
    let rp = GOLDILOCKS_POSEIDON2_PARTIAL_ROUNDS_8; // 22
    first_primes(offset + rp)[offset..].iter()
        .take(rp).map(|&p| k_constant(p)).collect()
}

fn k_external_constants_12() -> ExternalLayerConstants<Goldilocks, 12> {
    let offset = 64; // past t=8 external constants
    let total = 2 * GOLDILOCKS_POSEIDON2_HALF_FULL_ROUNDS * 12; // 96
    let vals: Vec<Goldilocks> = first_primes(offset + total)[offset..]
        .iter().take(total).map(|&p| k_constant(p)).collect();
    make_external::<12>(&vals)
}

fn k_internal_constants_12() -> Vec<Goldilocks> {
    let offset = 64 + 2 * GOLDILOCKS_POSEIDON2_HALF_FULL_ROUNDS * 12; // 160
    let rp = GOLDILOCKS_POSEIDON2_PARTIAL_ROUNDS_12; // 22
    first_primes(offset + rp)[offset..].iter()
        .take(rp).map(|&p| k_constant(p)).collect()
}

/// ci-poseidon t=8 with K-sequence (Ci=85/27) constants.
pub fn ci_poseidon2_k_8() -> Poseidon2Goldilocks<8> {
    Poseidon2::new(k_external_constants_8(), k_internal_constants_8())
}

/// ci-poseidon t=12 with K-sequence (Ci=85/27) constants.
pub fn ci_poseidon2_k_12() -> Poseidon2Goldilocks<12> {
    Poseidon2::new(k_external_constants_12(), k_internal_constants_12())
}

// ── tHz permutation constructors ─────────────────────────────────────────────

/// Generate external constants by cycling through tHz anchor values.
/// Each round uses a cyclic shift of the tHz array — full coverage,
/// no repetition within a round, structured harmonic rotation.
fn thz_external_constants_8() -> ExternalLayerConstants<Goldilocks, 8> {
    let half_rf = GOLDILOCKS_POSEIDON2_HALF_FULL_ROUNDS; // 4
    let mut vals = Vec::with_capacity(2 * half_rf * 8);
    for r in 0..2*half_rf {
        for i in 0..8 {
            // Cycle with round offset for diffusion across rounds
            vals.push(thz_constant(THZ_T8[(i + r) % 8]));
        }
    }
    make_external::<8>(&vals)
}

fn thz_internal_constants_8() -> Vec<Goldilocks> {
    let rp = GOLDILOCKS_POSEIDON2_PARTIAL_ROUNDS_8; // 22
    // Internal constants cycle through tHz values with nm complement alternating
    (0..rp).map(|i| {
        let thz = THZ_T8[i % 8];
        let nm = 11070 - thz; // complement: tHz + nm = 1107.0 × 10
        if i % 2 == 0 { thz_constant(thz) } else { thz_constant(nm) }
    }).collect()
}

fn thz_external_constants_12() -> ExternalLayerConstants<Goldilocks, 12> {
    let half_rf = GOLDILOCKS_POSEIDON2_HALF_FULL_ROUNDS;
    let mut vals = Vec::with_capacity(2 * half_rf * 12);
    for r in 0..2*half_rf {
        for i in 0..12 {
            vals.push(thz_constant(THZ_T12[(i + r) % 12]));
        }
    }
    make_external::<12>(&vals)
}

fn thz_internal_constants_12() -> Vec<Goldilocks> {
    let rp = GOLDILOCKS_POSEIDON2_PARTIAL_ROUNDS_12; // 22
    (0..rp).map(|i| {
        let thz = THZ_T12[i % 12];
        let nm = 11070 - thz;
        if i % 2 == 0 { thz_constant(thz) } else { thz_constant(nm) }
    }).collect()
}

/// ci-poseidon t=8 with periodic table (tHz) constants — cols 3-10 (Be→Na).
/// Pairs sum to 1107 tHz — mirror symmetry.
pub fn ci_poseidon2_thz_8() -> Poseidon2Goldilocks<8> {
    Poseidon2::new(thz_external_constants_8(), thz_internal_constants_8())
}

/// ci-poseidon t=12 with periodic table (tHz) constants — cols 1-12 (He→Al).
/// All pairs sum to 1107 tHz — same balance constant as t=8.
pub fn ci_poseidon2_thz_12() -> Poseidon2Goldilocks<12> {
    Poseidon2::new(thz_external_constants_12(), thz_internal_constants_12())
}

// ── Avalanche measurement ─────────────────────────────────────────────────────

pub fn avalanche<const WIDTH: usize>(
    perm: &impl Permutation<[Goldilocks; WIDTH]>,
    trials: usize,
) -> f64 {
    let primes = first_primes(512);
    let mut changes = 0u64;
    let total_bits = (trials * WIDTH * 64) as u64;
    for t in 0..trials {
        let mut state = [Goldilocks::ZERO; WIDTH];
        for i in 0..WIDTH {
            state[i] = k_constant(primes[(t * WIDTH + i) % 512]);
        }
        let mut out1 = state;
        perm.permute_mut(&mut out1);
        let mut state2 = state;
        state2[0] = Goldilocks::new(state[0].as_canonical_u64() ^ 1);
        let mut out2 = state2;
        perm.permute_mut(&mut out2);
        for i in 0..WIDTH {
            changes += (out1[i].as_canonical_u64() ^ out2[i].as_canonical_u64()).count_ones() as u64;
        }
    }
    changes as f64 / total_bits as f64 * 100.0
}

// ── Tests ─────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_thz_symmetry_property() {
        // Verify the 1107 tHz balance constant holds for both widths
        for i in 0..4 {
            assert_eq!(THZ_T8[i] + THZ_T8[7-i], 11070,
                "t=8 pair {} symmetry broken", i);
        }
        for i in 0..6 {
            assert_eq!(THZ_T12[i] + THZ_T12[11-i], 11070,
                "t=12 pair {} symmetry broken", i);
        }
        println!("✓ tHz balance constant 1107.0 holds for both t=8 and t=12");
        println!("  t=8  pairs: Be+Na=1107, B+Ne=1107, C+F=1107, N+O=1107");
        println!("  t=12 pairs: He+Al=1107, Li+Mg=1107, Be+Na=1107, B+Ne=1107,");
        println!("              C+F=1107, N+O=1107");
    }

    #[test]
    fn test_thz_decreasing_by_30() {
        // tHz decreases by exactly 30 per column (300 × 10)
        for i in 0..11 {
            assert_eq!(THZ_T12[i] - THZ_T12[i+1], 300,
                "tHz step not 30 at col {}", i+1);
        }
        println!("✓ tHz decreases by exactly 30 per column across all 12");
    }

    #[test]
    fn test_k_constants_nonzero_and_distinct() {
        let constants = generate_k_constants(30);
        for (i, c) in constants.iter().enumerate() {
            assert_ne!(c.as_canonical_u64(), 0, "K[{}] is zero", i);
        }
        for i in 0..constants.len() {
            for j in (i+1)..constants.len() {
                assert_ne!(constants[i].as_canonical_u64(),
                    constants[j].as_canonical_u64(), "K[{}]==K[{}]", i, j);
            }
        }
    }

    #[test]
    fn test_all_permutations_deterministic() {
        let perms: Vec<(&str, Box<dyn Fn() -> bool>)> = vec![
            ("k_8",   Box::new(|| {
                let p = ci_poseidon2_k_8();
                let s: [Goldilocks; 8] = core::array::from_fn(|i| Goldilocks::new(i as u64 + 1));
                let mut o1 = s; p.permute_mut(&mut o1);
                let mut o2 = s; p.permute_mut(&mut o2);
                o1 == o2 && o1[0].as_canonical_u64() != 1
            })),
            ("thz_8",  Box::new(|| {
                let p = ci_poseidon2_thz_8();
                let s: [Goldilocks; 8] = core::array::from_fn(|i| Goldilocks::new(i as u64 + 1));
                let mut o1 = s; p.permute_mut(&mut o1);
                let mut o2 = s; p.permute_mut(&mut o2);
                o1 == o2 && o1[0].as_canonical_u64() != 1
            })),
            ("k_12",  Box::new(|| {
                let p = ci_poseidon2_k_12();
                let s: [Goldilocks; 12] = core::array::from_fn(|i| Goldilocks::new(i as u64 + 1));
                let mut o1 = s; p.permute_mut(&mut o1);
                let mut o2 = s; p.permute_mut(&mut o2);
                o1 == o2 && o1[0].as_canonical_u64() != 1
            })),
            ("thz_12", Box::new(|| {
                let p = ci_poseidon2_thz_12();
                let s: [Goldilocks; 12] = core::array::from_fn(|i| Goldilocks::new(i as u64 + 1));
                let mut o1 = s; p.permute_mut(&mut o1);
                let mut o2 = s; p.permute_mut(&mut o2);
                o1 == o2 && o1[0].as_canonical_u64() != 1
            })),
        ];
        for (name, check) in &perms {
            assert!(check(), "{} permutation failed determinism check", name);
            println!("✓ {} deterministic", name);
        }
    }

    #[test]
    fn test_all_differ_from_default() {
        let k8  = ci_poseidon2_k_8();
        let t8  = ci_poseidon2_thz_8();
        let def = default_goldilocks_poseidon2_8();
        let s: [Goldilocks; 8] = core::array::from_fn(|i| Goldilocks::new(i as u64 + 1));
        let mut ko = s; k8.permute_mut(&mut ko);
        let mut to = s; t8.permute_mut(&mut to);
        let mut dfo = s; def.permute_mut(&mut dfo);
        assert_ne!(ko, dfo, "K-sequence should differ from LFSR default");
        assert_ne!(to, dfo, "tHz should differ from LFSR default");
        assert_ne!(ko, to,  "K-sequence should differ from tHz");
        println!("✓ All three constant sets produce distinct outputs");
    }

    #[test]
    fn test_summary() {
        let k8   = ci_poseidon2_k_8();
        let thz8  = ci_poseidon2_thz_8();
        let k12  = ci_poseidon2_k_12();
        let thz12 = ci_poseidon2_thz_12();
        let def8  = default_goldilocks_poseidon2_8();
        let def12 = default_goldilocks_poseidon2_12();

        let trials = 500;
        let av_k8    = avalanche::<8>(&k8,    trials);
        let av_thz8  = avalanche::<8>(&thz8,  trials);
        let av_def8  = avalanche::<8>(&def8,  trials);
        let av_k12   = avalanche::<12>(&k12,   trials);
        let av_thz12 = avalanche::<12>(&thz12, trials);
        let av_def12 = avalanche::<12>(&def12, trials);

        println!("\n╔══════════════════════════════════════════════════════════════╗");
        println!("║  ci-poseidon STARK — Plonky3/Goldilocks Avalanche Summary    ║");
        println!("║  Field: Goldilocks   S-box: x^7   rf=8 rp=22   June 2026    ║");
        println!("╠══════════════════════════════════════════════════════════════╣");
        println!("║  Width  Constants            Avalanche   Derivation          ║");
        println!("║  ─────  ──────────────────   ─────────   ──────────────────  ║");
        println!("║  t=8    K-sequence (85/27)   {:.2}%      rational constant  ║", av_k8);
        println!("║  t=8    tHz cols 3-10         {:.2}%      periodic table    ║", av_thz8);
        println!("║  t=8    Grain LFSR (default)  {:.2}%      pseudorandom      ║", av_def8);
        println!("╠══════════════════════════════════════════════════════════════╣");
        println!("║  t=12   K-sequence (85/27)   {:.2}%      rational constant  ║", av_k12);
        println!("║  t=12   tHz cols 1-12         {:.2}%      periodic table    ║", av_thz12);
        println!("║  t=12   Grain LFSR (default)  {:.2}%      pseudorandom      ║", av_def12);
        println!("╠══════════════════════════════════════════════════════════════╣");
        println!("║  tHz balance constant: 1107.0 tHz at both t=8 and t=12      ║");
        println!("║  t=8:  Be+Na=B+Ne=C+F=N+O=1107 (cols 3-10, mirror sym)     ║");
        println!("║  t=12: He+Al=...=N+O=1107 (all 12 cols, full symmetry)      ║");
        println!("╚══════════════════════════════════════════════════════════════╝");

        assert!(av_k8   > 40.0 && av_k8   < 60.0, "K t=8 avalanche out of range");
        assert!(av_thz8  > 40.0 && av_thz8  < 60.0, "tHz t=8 avalanche out of range");
        assert!(av_k12  > 40.0 && av_k12  < 60.0, "K t=12 avalanche out of range");
        assert!(av_thz12 > 40.0 && av_thz12 < 60.0, "tHz t=12 avalanche out of range");
    }
}
