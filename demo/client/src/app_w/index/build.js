import fs from "fs";
import { BuildPathMaker } from "../../../lib/builder";

const bPath = new BuildPathMaker("./src/app_w/index", "./dist/app_w/index");

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

await Bun.build({
  entrypoints: [bPath.src("sw.js")],
  outdir: bPath.dist(""),
  minify: {
    whitespace: true,
    identifiers: true,
    syntax: true,
    properties: true,
    keepNames: false
  },
  drop: ["console"],
  sourcemap: "none",
  target: "browser",
  splitting: false
});


fs.copyFileSync(...bPath.src_dist("index.html"));
fs.copyFileSync(...bPath.src_dist("site.webmanifest"));
fs.copyFileSync(bPath.src("assets/web-app-manifest-192x192.png"), bPath.dist("web-app-manifest-192x192.png"));
fs.copyFileSync(bPath.src("assets/web-app-manifest-512x512.png"), bPath.dist("web-app-manifest-512x512.png"));
fs.copyFileSync(bPath.src("assets/screenshot-desktop-1.png"), bPath.dist("screenshot-desktop-1.png"));
fs.copyFileSync(bPath.src("assets/screenshot-mobile-1.png"), bPath.dist("screenshot-mobile-1.png"));