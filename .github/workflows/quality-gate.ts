import { spawnSync } from "bun";

export type Counts = { functions: number; quality: number; patterns: number };
type Baseline = Counts & { version: 1; gstyle_version: string };

export function checkRates(current: Counts, baseline: Counts): string[] {
  const failures: string[] = [];
  for (const metric of ["quality", "patterns"] as const) {
    if (current[metric] * baseline.functions > baseline[metric] * current.functions) {
      failures.push(`${metric} rate exceeds baseline`);
    }
  }
  return failures;
}

export function validateCounts(counts: Counts): void {
  for (const key of ["functions", "quality", "patterns"] as const) {
    if (!Number.isSafeInteger(counts[key]) || counts[key] < (key === "functions" ? 1 : 0)) {
      throw new Error(`invalid ${key} count`);
    }
  }
}

function scan(command: string[]): any {
  const result = spawnSync([process.env.GSTYLE ?? "gstyle", ...command], { stdout: "pipe", stderr: "pipe" });
  // check exits 1 for findings; an analysis failure is never a passing scan.
  if (result.exitCode !== 0 && !(command[0] === "check" && command.includes("json") && result.exitCode === 1)) {
    throw new Error(`gstyle ${command[0]} failed: ${result.stderr.toString()}`);
  }
  return JSON.parse(result.stdout.toString());
}

async function main(): Promise<void> {
  const baselineFile = Bun.file("quality-baseline.json");
  const baseline: Baseline = await baselineFile.json();
  if (baseline.version !== 1 || !/^[a-f0-9]{40}$/.test(baseline.gstyle_version)) {
    throw new Error("invalid baseline version or Gstyle revision");
  }
  validateCounts(baseline);
  // Enforce the 200-code-line cap from facts: suppressed diagnostics omit violations.
  const facts = scan(["check", "--format", "facts", "."]);
  if (facts.schema_version !== 2 || !Array.isArray(facts.functions)) {
    throw new Error("invalid Gstyle facts");
  }
  for (const fn of facts.functions) {
    if (!Number.isSafeInteger(fn.code_lines) || fn.code_lines < 0) {
      throw new Error("invalid Gstyle function length");
    }
    if (fn.code_lines > 200) {
      throw new Error("200-line function limit exceeded");
    }
  }
  const quality = scan(["check", "--format", "json", "."]);
  const patterns = scan(["patterns", "--format", "json", "."]);
  if (quality.schema_version !== 1 || !Array.isArray(quality.diagnostics) || !Array.isArray(patterns.candidates)) {
    throw new Error("invalid Gstyle report");
  }
  const current = {
    functions: quality.summary?.functions,
    quality: quality.diagnostics.length,
    patterns: patterns.candidates.length,
  };
  validateCounts(current);
  for (const metric of ["quality", "patterns"] as const) {
    const rate = (count: Counts) => (count[metric] * 1000 / count.functions).toFixed(2);
    console.log(`${metric}: ${current[metric]} / ${current.functions} functions = ${rate(current)} per 1000; baseline ${rate(baseline)}`);
  }
  if (process.argv.includes("--write")) {
    await Bun.write(baselineFile, JSON.stringify({ ...baseline, ...current }, null, 2) + "\n");
    console.log("Baseline updated. Review this change before committing.");
    return;
  }
  const failures = checkRates(current, baseline);
  if (failures.length) throw new Error(failures.join("; "));
  console.log("Quality gate passed.");
}

if (import.meta.main) {
  main().catch((error: Error) => {
    console.error(error.message);
    process.exitCode = 1;
  });
}
