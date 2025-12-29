import fs from "fs";
import path from "path";

/**
 * Multi-Page Application (MPA) Builder for WBX Demo
 * Builds all apps in src/ directory
 */

const APPS = [
  { name: "ws-test", entry: "app.js", hasCSS: true },
  { name: "sse-test", entry: "app.js", hasCSS: true },
  // Add more apps here as needed
];

async function buildApp(app) {
  const srcDir = `./src/${app.name}`;
  const outDir = `./dist/${app.name}`;

  console.log(`\n📦 Building ${app.name}...`);

  // Ensure output directory exists
  if (!fs.existsSync(outDir)) {
    fs.mkdirSync(outDir, { recursive: true });
  }

  // Build JavaScript
  const entryPath = path.join(srcDir, app.entry);
  if (fs.existsSync(entryPath)) {
    await Bun.build({
      entrypoints: [entryPath],
      outdir: outDir,
      naming: "[dir]/[name].[ext]",
      minify: {
        whitespace: true,
        identifiers: true,
        syntax: true,
        properties: true,
        keepNames: false
      },
      sourcemap: "none",
      target: "browser",
      splitting: false,
    });
    console.log(`  ✓ Bundled ${app.entry}`);
  }

  // Copy HTML
  const htmlPath = path.join(srcDir, "index.html");
  if (fs.existsSync(htmlPath)) {
    fs.copyFileSync(htmlPath, path.join(outDir, "index.html"));
    console.log(`  ✓ Copied index.html`);
  }

  // Copy CSS
  if (app.hasCSS) {
    const cssPath = path.join(srcDir, "styles.css");
    if (fs.existsSync(cssPath)) {
      fs.copyFileSync(cssPath, path.join(outDir, "styles.css"));
      console.log(`  ✓ Copied styles.css`);
    }
  }

  // Copy any additional assets
  const assetsDir = path.join(srcDir, "assets");
  if (fs.existsSync(assetsDir)) {
    const outAssetsDir = path.join(outDir, "assets");
    fs.mkdirSync(outAssetsDir, { recursive: true });
    const files = fs.readdirSync(assetsDir);
    files.forEach(file => {
      fs.copyFileSync(
        path.join(assetsDir, file),
        path.join(outAssetsDir, file)
      );
    });
    console.log(`  ✓ Copied ${files.length} asset(s)`);
  }
}

// Build all apps
console.log("🚀 Building MPA apps...");
for (const app of APPS) {
  await buildApp(app);
}

console.log("\n✅ All apps built successfully!\n");
