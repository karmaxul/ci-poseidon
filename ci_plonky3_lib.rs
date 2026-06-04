//! ci-poseidon STARK variant — Plonky3/Goldilocks implementation
//!
//! Implements the Ci=85/27 constant framework as a drop-in replacement for
//! Plonky3's default Grain LFSR constants in Poseidon2 over Goldilocks.
//!
//! Supported widths: t=8, t=12 (standard Plonky3 Merkle tree widths)
//! Field: Goldilocks (p = 2^64 - 2^32 + 1)
//! S-box: x^7 (Goldilocks standard, gcd(7, p-1) = 1)
//!
//! The key research question: do ci-derived constants produce comparable
//! avalanche and security properties to Grain LFSR constants, while being
//! verifiable from first principles (no LFSR seed trust)?
//!
//! Author:  Christopher Seekins — Harmony Worldwide / HealChain
//! Date:    June 2026

use p3_field::{Field, PrimeCharacteristicRing, PrimeField64};
use p3_goldilocks::Goldilocks;
use p3_goldilocks::poseidon2::{
    Poseidon2ExternalLayerGoldilocks, Poseidon2InternalLayerGoldilocks,
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

// ── K-constant generation over Goldilocks ────────────────────────────────────

/// Sieve the first `n` primes.
fn first_primes(n: usize) -> Vec<u64> {
    let n = n.min(512);
    let mut primes = Vec::with_capacity(n);
    let mut sieve = vec![true; 3700];
    sieve[0] = false;
    sieve[1] = false;
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

/// Fast modular exponentiation.
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

/// K[i] = (85 * prime[i] * 2^64) * inv(27 * (prime[i]+1)) mod p (Goldilocks)
///
/// 2^64 mod p = 2^32 - 1 = 4294967295 (since p = 2^64 - 2^32 + 1)
pub fn k_constant(prime: u64) -> Goldilocks {
    let p = GOLDILOCKS_PRIME as u128;
    let pow64_mod_p: u128 = 4_294_967_295;
    let num = (85u128 * prime as u128 % p) * pow64_mod_p % p;
    let denom = (27u128 * (prime as u128 + 1)) % p;
    let denom_inv = mod_pow(denom, p - 2, p);
    Goldilocks::from_canonical_u64((num * denom_inv % p) as u64)
}

/// Generate `count` K-constants over Goldilocks from the prime sequence.
pub fn generate_k_constants(count: usize) -> Vec<Goldilocks> {
    first_primes(count).iter().map(|&p| k_constant(p)).collect()
}

// ── ci-poseidon constant builders ─────────────────────────────────────────────

/// Build ExternalLayerConstants for width WIDTH using ci-derived K-constants.
/// Offset into the K-sequence by width to ensure distinct constants per width.
fn ci_external_constants<const WIDTH: usize>() -> ExternalLayerConstants<Goldilocks, WIDTH> {
    let half_rf = GOLDILOCKS_POSEIDON2_HALF_FULL_ROUNDS; // 4
    let total = 2 * half_rf * WIDTH;
    let offset = match WIDTH { 8 => 0, 12 => 64, 16 => 160, _ => 0 };

    let primes = first_primes(offset + total);
    let consts: Vec<Goldilocks> = primes[offset..].iter()
        .take(total)
        .map(|&p| k_constant(p))
        .collect();

    let mut initial: Vec<[Goldilocks; WIDTH]> = Vec::with_capacity(half_rf);
    let mut terminal: Vec<[Goldilocks; WIDTH]> = Vec::with_capacity(half_rf);

    for r in 0..half_rf {
        let mut arr = [Goldilocks::ZERO; WIDTH];
        for i in 0..WIDTH { arr[i] = consts[r * WIDTH + i]; }
        initial.push(arr);
    }
    for r in 0..half_rf {
        let mut arr = [Goldilocks::ZERO; WIDTH];
        for i in 0..WIDTH { arr[i] = consts[(half_rf + r) * WIDTH + i]; }
        terminal.push(arr);
    }

    ExternalLayerConstants::new(initial, terminal)
}

/// Build internal constants for width 8 (rp=22 constants).
fn ci_internal_constants_8() -> Vec<Goldilocks> {
    let offset = 0 + 2 * GOLDILOCKS_POSEIDON2_HALF_FULL_ROUNDS * 8; // 64
    let rp = GOLDILOCKS_POSEIDON2_PARTIAL_ROUNDS_8; // 22
    first_primes(offset + rp)[offset..].iter()
        .take(rp).map(|&p| k_constant(p)).collect()
}

/// Build internal constants for width 12 (rp=22 constants).
fn ci_internal_constants_12() -> Vec<Goldilocks> {
    let offset = 64 + 2 * GOLDILOCKS_POSEIDON2_HALF_FULL_ROUNDS * 12; // 160
    let rp = GOLDILOCKS_POSEIDON2_PARTIAL_ROUNDS_12; // 22
    first_primes(offset + rp)[offset..].iter()
        .take(rp).map(|&p| k_constant(p)).collect()
}

// ── Permutation constructors ──────────────────────────────────────────────────

/// ci-poseidon over Goldilocks, width 8.
/// Drop-in for default_goldilocks_poseidon2_8() with ci-derived constants.
pub fn ci_poseidon2_goldilocks_8() -> Poseidon2Goldilocks<8> {
    Poseidon2::new(ci_external_constants::<8>(), ci_internal_constants_8())
}

/// ci-poseidon over Goldilocks, width 12.
/// Drop-in for default_goldilocks_poseidon2_12() with ci-derived constants.
pub fn ci_poseidon2_goldilocks_12() -> Poseidon2Goldilocks<12> {
    Poseidon2::new(ci_external_constants::<12>(), ci_internal_constants_12())
}

// ── Avalanche measurement ─────────────────────────────────────────────────────

/// Flip bit 0 of element 0, count differing output bits. Returns % changed.
pub fn avalanche<const WIDTH: usize>(
    perm: &impl Permutation<[Goldilocks; WIDTH]>,
    trials: usize,
) -> f64 {
    let primes = first_primes(trials * WIDTH + 2);
    let mut changes = 0u64;
    let total_bits = (trials * WIDTH * 64) as u64;

    for t in 0..trials {
        let mut state = [Goldilocks::ZERO; WIDTH];
        for i in 0..WIDTH {
            state[i] = k_constant(primes[t * WIDTH + i]);
        }
        let mut out1 = state;
        perm.permute_mut(&mut out1);

        let mut state2 = state;
        state2[0] = Goldilocks::from_canonical_u64(state[0].as_canonical_u64() ^ 1);
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
    fn test_k_constants_nonzero_and_distinct() {
        let constants = generate_k_constants(30);
        for (i, c) in constants.iter().enumerate() {
            assert_ne!(c.as_canonical_u64(), 0, "K[{}] is zero", i);
        }
        for i in 0..constants.len() {
            for j in (i+1)..constants.len() {
                assert_ne!(
                    constants[i].as_canonical_u64(),
                    constants[j].as_canonical_u64(),
                    "K[{}] == K[{}]", i, j
                );
            }
        }
        println!("K[0] = {:#018x}", constants[0].as_canonical_u64());
        println!("K[1] = {:#018x}", constants[1].as_canonical_u64());
        println!("K[2] = {:#018x}", constants[2].as_canonical_u64());
    }

    #[test]
    fn test_ci_poseidon2_t8_deterministic() {
        let perm = ci_poseidon2_goldilocks_8();
        let state: [Goldilocks; 8] = core::array::from_fn(|i| {
            Goldilocks::from_canonical_u64(i as u64 + 1)
        });
        let mut out1 = state;
        perm.permute_mut(&mut out1);
        let mut out2 = state;
        perm.permute_mut(&mut out2);
        assert_eq!(out1, out2, "ci permutation t=8 not deterministic");
        assert_ne!(out1[0].as_canonical_u64(), 1, "output == input");
        println!("ci-poseidon t=8 output[0]: {:#018x}", out1[0].as_canonical_u64());
    }

    #[test]
    fn test_ci_poseidon2_t12_deterministic() {
        let perm = ci_poseidon2_goldilocks_12();
        let state: [Goldilocks; 12] = core::array::from_fn(|i| {
            Goldilocks::from_canonical_u64(i as u64 + 1)
        });
        let mut out1 = state;
        perm.permute_mut(&mut out1);
        let mut out2 = state;
        perm.permute_mut(&mut out2);
        assert_eq!(out1, out2, "ci permutation t=12 not deterministic");
        println!("ci-poseidon t=12 output[0]: {:#018x}", out1[0].as_canonical_u64());
    }

    #[test]
    fn test_ci_differs_from_default() {
        // ci-poseidon constants should produce different output than Grain LFSR
        let ci = ci_poseidon2_goldilocks_8();
        let default = default_goldilocks_poseidon2_8();
        let state: [Goldilocks; 8] = core::array::from_fn(|i| {
            Goldilocks::from_canonical_u64(i as u64 + 1)
        });
        let mut ci_out = state;
        ci.permute_mut(&mut ci_out);
        let mut default_out = state;
        default.permute_mut(&mut default_out);
        assert_ne!(ci_out, default_out,
            "ci and default should produce different outputs (different constants)");
        println!("ci output[0]:      {:#018x}", ci_out[0].as_canonical_u64());
        println!("default output[0]: {:#018x}", default_out[0].as_canonical_u64());
    }

    #[test]
    fn test_avalanche_t8() {
        let perm = ci_poseidon2_goldilocks_8();
        let av = avalanche::<8>(&perm, 200);
        println!("Avalanche (ci-poseidon, Goldilocks, t=8): {:.2}%", av);
        assert!(av > 40.0 && av < 60.0,
            "Avalanche {:.2}% out of range 40-60%", av);
    }

    #[test]
    fn test_avalanche_t12() {
        let perm = ci_poseidon2_goldilocks_12();
        let av = avalanche::<12>(&perm, 200);
        println!("Avalanche (ci-poseidon, Goldilocks, t=12): {:.2}%", av);
        assert!(av > 40.0 && av < 60.0,
            "Avalanche {:.2}% out of range 40-60%", av);
    }

    #[test]
    fn test_avalanche_default_comparison() {
        // Compare ci vs default avalanche — both should be near 50%
        let ci8 = ci_poseidon2_goldilocks_8();
        let def8 = default_goldilocks_poseidon2_8();
        let ci_av = avalanche::<8>(&ci8, 500);
        let def_av = avalanche::<8>(&def8, 500);
        println!("\n╔══════════════════════════════════════════════════════╗");
        println!("║  Avalanche Comparison — Goldilocks t=8, 500 trials  ║");
        println!("╠══════════════════════════════════════════════════════╣");
        println!("║  ci-poseidon (Ci=85/27):  {:.2}%                   ║", ci_av);
        println!("║  Default (Grain LFSR):    {:.2}%                   ║", def_av);
        println!("║  Delta:                   {:.2}%                   ║", (ci_av - def_av).abs());
        println!("╚══════════════════════════════════════════════════════╝");
        assert!(ci_av > 40.0 && ci_av < 60.0);
        assert!(def_av > 40.0 && def_av < 60.0);
    }

    #[test]
    fn test_summary() {
        let ci8  = ci_poseidon2_goldilocks_8();
        let ci12 = ci_poseidon2_goldilocks_12();
        let def8  = default_goldilocks_poseidon2_8();
        let def12 = default_goldilocks_poseidon2_12();

        let ci_av8   = avalanche::<8>(&ci8,   500);
        let ci_av12  = avalanche::<12>(&ci12,  500);
        let def_av8  = avalanche::<8>(&def8,   500);
        let def_av12 = avalanche::<12>(&def12, 500);

        println!("\n╔══════════════════════════════════════════════════════════╗");
        println!("║  ci-poseidon STARK — Plonky3/Goldilocks Summary          ║");
        println!("║  June 2026                                               ║");
        println!("╠══════════════════════════════════════════════════════════╣");
        println!("║  Field: Goldilocks (p = 2^64 - 2^32 + 1)                ║");
        println!("║  S-box: x^7   rf=8   rp=22   (Plonky3 standard)         ║");
        println!("╠══════════════════════════════════════════════════════════╣");
        println!("║  Width  Constants     Avalanche (ci)  Avalanche (LFSR)  ║");
        println!("║  t=8    Ci=85/27      {:.2}%          {:.2}%          ║", ci_av8,  def_av8);
        println!("║  t=12   Ci=85/27      {:.2}%          {:.2}%          ║", ci_av12, def_av12);
        println!("╠══════════════════════════════════════════════════════════╣");
        println!("║  K[0] = {:#018x}                        ║", generate_k_constants(1)[0].as_canonical_u64());
        println!("║  Verifiable: K[i] = (85*p_i*2^64)*inv(27*(p_i+1)) mod p ║");
        println!("║  No LFSR seed trust required                             ║");
        println!("╚══════════════════════════════════════════════════════════╝");
    }
}
