import { expect, test } from "bun:test";
import { checkRates, validateCounts } from "./quality-gate";

const baseline = { functions: 164, quality: 46, patterns: 77 };

test("equal rates pass when the codebase grows", () => {
  expect(checkRates({ functions: 328, quality: 92, patterns: 154 }, baseline)).toEqual([]);
});

test("either rate can fail independently", () => {
  expect(checkRates({ ...baseline, quality: 47 }, baseline)).toEqual(["quality rate exceeds baseline"]);
  expect(checkRates({ ...baseline, patterns: 78 }, baseline)).toEqual(["patterns rate exceeds baseline"]);
});

test("a smaller denominator can expose a regression", () => {
  expect(checkRates({ ...baseline, functions: 163 }, baseline)).toHaveLength(2);
});

test("lower rates and a zero baseline work", () => {
  expect(checkRates({ ...baseline, quality: 45, patterns: 76 }, baseline)).toEqual([]);
  expect(checkRates({ functions: 1, quality: 1, patterns: 0 }, { functions: 1, quality: 0, patterns: 0 })).toEqual(["quality rate exceeds baseline"]);
});

test("empty and malformed scans fail", () => {
  for (const invalid of [0, -1, NaN, Infinity, 1.5, undefined]) {
    expect(() => validateCounts({ ...baseline, functions: invalid as number })).toThrow();
  }
  expect(() => validateCounts({ ...baseline, quality: -1 })).toThrow();
  expect(() => validateCounts({ ...baseline, patterns: NaN })).toThrow();
  expect(() => validateCounts({ functions: 1, quality: 0, patterns: 0 })).not.toThrow();
});

// Exercise the CLI with scan results rather than relying only on rate arithmetic.
test("CLI rejects failed analysis and hard-limit violations", async () => {
  const { mkdtempSync, writeFileSync, rmSync } = await import("node:fs");
  const { tmpdir } = await import("node:os");
  const { join } = await import("node:path");
  const { fileURLToPath } = await import("node:url");
  const directory = mkdtempSync(join(tmpdir(), "openrpc-quality-gate-"));
  const script = fileURLToPath(new URL("./quality-gate.ts", import.meta.url));
  try {
    writeFileSync(join(directory, "quality-baseline.json"), JSON.stringify({ ...baseline, version: 1, cyclo_version: "a".repeat(40) }));
    for (const scenario of [
      { code: 1, diagnostics: [], functions: 164, error: "" },
      { code: 2, diagnostics: [], functions: 164, error: "cyclo check failed" },
      { code: 1, diagnostics: [{ rule_id: "fn_length" }], functions: 164, lines: 201, error: "200-line function limit exceeded" },
      { code: 0, diagnostics: [], functions: 0, error: "invalid functions count" },
      { code: 0, diagnostics: [], functions: 164, lines: 200, error: "" },
      { code: 0, diagnostics: [], functions: 164, lines: 201, error: "200-line function limit exceeded" },
      { code: 0, diagnostics: [], functions: 164, lines: -1, error: "invalid Cyclo function length" },
    ]) {
      const report = { schema_version: 1, summary: { functions: scenario.functions }, diagnostics: scenario.diagnostics };
      const facts = { schema_version: 2, functions: [{ code_lines: scenario.lines ?? 20 }] };
      const executable = join(directory, "cyclo");
      writeFileSync(executable, `#!/usr/bin/env bun\nconst check = process.argv[2] === "check"; const facts = process.argv.includes("facts"); console.log(JSON.stringify(check ? (facts ? ${JSON.stringify(facts)} : ${JSON.stringify(report)}) : {candidates: []})); process.exit(check ? (facts && ${scenario.code} === 1 ? 0 : ${scenario.code}) : 0);\n`, { mode: 0o755 });
      const result = Bun.spawnSync([process.execPath, script], { cwd: directory, env: { ...process.env, CYCLO: executable } });
      expect(result.exitCode).toBe(scenario.error ? 1 : 0);
      if (scenario.error) expect(result.stderr.toString()).toContain(scenario.error);
    }
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});
