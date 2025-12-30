import fs from "fs";


await Bun.build({
  entrypoints: ["./src/index/app.js"],
  outdir: "./dist/index",
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
  entrypoints: ["./src/index/sw.js"],
  outdir: "./dist/index",
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


fs.copyFileSync("./src/index/index.html", "./dist/index/index.html");
fs.copyFileSync("./src/index/site.webmanifest", "./dist/index/site.webmanifest");
fs.copyFileSync("./src/index/assets/web-app-manifest-192x192.png", "./dist/index/web-app-manifest-192x192.png");
fs.copyFileSync("./src/index/assets/web-app-manifest-512x512.png", "./dist/index/web-app-manifest-512x512.png");
fs.copyFileSync("./src/index/assets/screenshot-desktop-1.png", "./dist/index/screenshot-desktop-1.png");
fs.copyFileSync("./src/index/assets/screenshot-mobile-1.png", "./dist/index/screenshot-mobile-1.png");