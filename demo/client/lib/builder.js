import { IndexBuilder } from "./build_utils/index_builder";

class BuildPathMaker {
  constructor(srcPathBase, distPathBase) {
    this.srcPathBase = srcPathBase;
    this.distPathBase = distPathBase;
  }
  src(relativePath) {
    return `${this.srcPathBase}/${relativePath}`;
  }
  dist(relativePath) {
    return `${this.distPathBase}/${relativePath}`;
  }
  src_dist(relativePath) {
    return [this.src(relativePath), this.dist(relativePath)];
  }
}

export { IndexBuilder, BuildPathMaker };