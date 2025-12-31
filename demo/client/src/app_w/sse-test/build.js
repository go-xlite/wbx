import fs from "fs";
import { transform } from 'lightningcss';
import { IndexBuilder, BuildPathMaker } from "../../../lib/builder.js";

const bPath = new BuildPathMaker("./src/app_w/sse-test", "./dist/app_w/sse-test");

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


const css = fs.readFileSync(bPath.src("styles.css"), 'utf8');
const { code } = transform({
  filename: 'style.css',
  code: Buffer.from(css),
  minify: true,
});

new IndexBuilder()
  .readHTML(bPath.src("index.html"))
  .embedCSS(code.toString())
  .embedJsFromFile("./lib-dist/embed.js")
  .writeToFile(bPath.dist("index.html"));
