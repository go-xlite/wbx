import fs from "fs";
import { transform } from 'lightningcss';
import { IndexBuilder, BuildPathMaker } from "../../../lib/builder.js";

const bPath = new BuildPathMaker("./src/app_g/login", "./dist/app_g/login");

await Bun.build({
  entrypoints: [bPath.src("app.js")],
  outdir: bPath.dist(""),
  minify: {
    whitespace: true,
    identifiers: true,
    syntax: true,
    properties: true,
    keepNames: false
  },
  sourcemap: "none",
  target: "browser",
  splitting: false
});

// Copy auth-manager.js to dist (not minified for debugging)
fs.copyFileSync(...bPath.src_dist("auth-manager.js"));

const css = fs.readFileSync(bPath.src("styles.css"), 'utf8');
const { code } = transform({
  filename: 'style.css',
  code: Buffer.from(css),
  minify: true,
});

new IndexBuilder()
  .readHTML(bPath.src("index.html"))
  .embedJsFromFile("./lib-dist/embed.js")
  .embedCSS(code.toString())
  .writeToFile(bPath.dist("index.html"));