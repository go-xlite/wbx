import fs from "fs";
import { transform } from 'lightningcss';


await Bun.build({
  entrypoints: ["./src/api-demo/app.js"],
  outdir: "./dist/api-demo",
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

// Read and minify CSS
const css = fs.readFileSync("./src/api-demo/styles.css", 'utf8');
const { code } = transform({
  filename: 'style.css',
  code: Buffer.from(css),
  minify: true,
});

// Read HTML and embed CSS
const html = fs.readFileSync("./src/api-demo/index.html", 'utf8');
const htmlWithEmbeddedCSS = html.replace(
  '<embedded-css></embedded-css>',
  `<style>${code.toString()}</style>`
);

fs.writeFileSync("./dist/api-demo/index.html", htmlWithEmbeddedCSS);