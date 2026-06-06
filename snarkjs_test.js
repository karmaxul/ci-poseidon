/**
 * snarkjs_test.js — ci-poseidon snarkjs verification test
 * Complements gnark Groth16 benchmarks with JS-side verification.
 * Uses the same circuits and zkeys as the browser integrity layer.
 *
 * Run: node snarkjs_test.js
 */

const snarkjs = require("snarkjs");
const fs = require("fs");
const path = require("path");

const CIRCUITS_DIR = path.join(process.env.HOME, "ci-sha-project/docs/circuits");
const BN254_P = 21888242871839275222246405745257275088548364400416034343698204186575808495617n;

function modpow(base, exp, mod) {
    let result = 1n;
    base = base % mod;
    while (exp > 0n) {
        if (exp % 2n === 1n) result = result * base % mod;
        exp = exp / 2n;
        base = base * base % mod;
    }
    return result;
}

function modinv(a, m) { return modpow(a, m - 2n, m); }

function kConstant(i, p) {
    const primes = [2n, 3n, 5n, 7n, 11n, 13n, 17n, 19n, 23n, 29n];
    const prime = primes[i % primes.length];
    return (85n * prime * (1n << 64n) % p * modinv(27n * (prime + 1n), p)) % p;
}

async function testWidth(t, in0, in1) {
    const wasmFile = path.join(CIRCUITS_DIR, `ci_poseidon_t${t}_bn254.wasm`);
    const zkeyFile = path.join(CIRCUITS_DIR, `ci_poseidon_t${t}_bn254_final.zkey`);
    const vkeyFile = path.join(CIRCUITS_DIR, `verification_key_t${t}.json`);

    if (!fs.existsSync(wasmFile) || !fs.existsSync(zkeyFile) || !fs.existsSync(vkeyFile)) {
        console.log(`  t=${t}: skipped (missing files)`);
        return null;
    }

    const inputSize = t === 2 || t === 3 ? 2 : t;
    const inputs = [in0.toString()];
    for (let i = 1; i < inputSize; i++) {
        inputs.push(kConstant(i, BN254_P).toString());
    }
    const input = { in: inputs };
    const start = Date.now();
    const { proof, publicSignals } = await snarkjs.groth16.fullProve(input, wasmFile, zkeyFile);
    const proveMs = Date.now() - start;

    const vkey = JSON.parse(fs.readFileSync(vkeyFile));
    const verifyStart = Date.now();
    const valid = await snarkjs.groth16.verify(vkey, publicSignals, proof);
    const verifyMs = Date.now() - verifyStart;

    return { t, valid, proveMs, verifyMs, commitment: BigInt(publicSignals[0]) };
}

async function main() {
    console.log("ci-poseidon snarkjs verification test");
    console.log("======================================");
    console.log(`Node.js ${process.version}, snarkjs 0.7.6`);
    console.log("");

    const in0 = 1n;
    const in1 = kConstant(1, BN254_P);

    console.log("Standard inputs: in[0]=1, in[1]=K[1]");
    console.log("");

    let allPass = true;
    for (const t of [2, 3, 4, 6]) {
        process.stdout.write(`  t=${t}: proving... `);
        const result = await testWidth(t, in0, in1);
        if (!result) { allPass = false; continue; }
        const status = result.valid ? "✓ PASS" : "✗ FAIL";
        if (!result.valid) allPass = false;
        console.log(`${status} | prove=${result.proveMs}ms verify=${result.verifyMs}ms | commitment=0x${result.commitment.toString(16).padStart(64,'0').slice(0,16)}...`);
    }

    console.log("");
    console.log(allPass ? "All widths: PASS ✓" : "Some widths FAILED ✗");

    console.log("");
    console.log("Determinism check (t=3, 3 runs):");
    const commitments = [];
    for (let i = 0; i < 3; i++) {
        const r = await testWidth(3, in0, in1);
        commitments.push(r.commitment);
        process.stdout.write(`  run ${i+1}: 0x${r.commitment.toString(16).slice(0,16)}...\n`);
    }
    const det = commitments.every(c => c === commitments[0]);
    console.log(`  Deterministic: ${det ? "YES ✓" : "NO ✗"}`);
}

main().catch(console.error);