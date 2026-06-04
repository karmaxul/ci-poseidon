use criterion::{criterion_group, criterion_main, Criterion};
use ci_plonky3::*;
use p3_goldilocks::Goldilocks;
use p3_symmetric::Permutation;

fn bench_k8(c: &mut Criterion) {
    let perm = ci_poseidon2_k_8();
    let state: [Goldilocks; 8] = core::array::from_fn(|i| Goldilocks::new(i as u64 + 1));
    c.bench_function("ci_poseidon_k_t8", |b| {
        b.iter(|| { let mut s = state; perm.permute_mut(&mut s); s })
    });
}

fn bench_thz8(c: &mut Criterion) {
    let perm = ci_poseidon2_thz_8();
    let state: [Goldilocks; 8] = core::array::from_fn(|i| Goldilocks::new(i as u64 + 1));
    c.bench_function("ci_poseidon_thz_t8", |b| {
        b.iter(|| { let mut s = state; perm.permute_mut(&mut s); s })
    });
}

fn bench_k12(c: &mut Criterion) {
    let perm = ci_poseidon2_k_12();
    let state: [Goldilocks; 12] = core::array::from_fn(|i| Goldilocks::new(i as u64 + 1));
    c.bench_function("ci_poseidon_k_t12", |b| {
        b.iter(|| { let mut s = state; perm.permute_mut(&mut s); s })
    });
}

fn bench_thz12(c: &mut Criterion) {
    let perm = ci_poseidon2_thz_12();
    let state: [Goldilocks; 12] = core::array::from_fn(|i| Goldilocks::new(i as u64 + 1));
    c.bench_function("ci_poseidon_thz_t12", |b| {
        b.iter(|| { let mut s = state; perm.permute_mut(&mut s); s })
    });
}

fn bench_default8(c: &mut Criterion) {
    use p3_goldilocks::default_goldilocks_poseidon2_8;
    let perm = default_goldilocks_poseidon2_8();
    let state: [Goldilocks; 8] = core::array::from_fn(|i| Goldilocks::new(i as u64 + 1));
    c.bench_function("default_lfsr_t8", |b| {
        b.iter(|| { let mut s = state; perm.permute_mut(&mut s); s })
    });
}

fn bench_default12(c: &mut Criterion) {
    use p3_goldilocks::default_goldilocks_poseidon2_12;
    let perm = default_goldilocks_poseidon2_12();
    let state: [Goldilocks; 12] = core::array::from_fn(|i| Goldilocks::new(i as u64 + 1));
    c.bench_function("default_lfsr_t12", |b| {
        b.iter(|| { let mut s = state; perm.permute_mut(&mut s); s })
    });
}

criterion_group!(benches, bench_k8, bench_thz8, bench_k12, bench_thz12, bench_default8, bench_default12);
criterion_main!(benches);
