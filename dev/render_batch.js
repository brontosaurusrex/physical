"use strict";

// Keep Canvas loops in JavaScript so a cache rebuild crosses the WASM boundary
// once, rather than once for every Canvas command. Commands and order match the
// Go renderer, including separate fill/stroke operations for each brick.
window.physicalCanvasBatch = {
  drawBricks(ctx, bytes, count, tilted, radius,
             normalFill, normalStroke, magicFill, magicStroke,
             unbreakableFill, unbreakableStroke) {
    const data = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
    const fills = [normalFill, magicFill, unbreakableFill];
    const strokes = [normalStroke, magicStroke, unbreakableStroke];

    for (let kind = 0; kind < 3; kind++) {
      ctx.fillStyle = fills[kind];
      ctx.strokeStyle = strokes[kind];
      ctx.lineWidth = kind === 0 ? 1 : 2;
      ctx.setLineDash([]);

      for (let i = 0; i < count; i++) {
        const offset = i * 48;
        const flags = data.getFloat64(offset + 40, true);
        const magic = (flags & 1) !== 0;
        const unbreakable = (flags & 2) !== 0;
        const matches = (kind === 0 && !magic && !unbreakable) ||
                        (kind === 1 && magic) ||
                        (kind === 2 && unbreakable);
        if (!matches) continue;

        const x = data.getFloat64(offset, true);
        const y = data.getFloat64(offset + 8, true);
        const w = data.getFloat64(offset + 16, true);
        const h = data.getFloat64(offset + 24, true);
        ctx.save();
        ctx.beginPath();
        if (tilted) {
          ctx.translate(x + w / 2, y + h / 2);
          ctx.rotate(data.getFloat64(offset + 32, true));
          ctx.roundRect(-w / 2, -h / 2, w, h, radius);
        } else {
          ctx.roundRect(x, y, w, h, radius);
        }
        ctx.fill();
        ctx.stroke();
        ctx.restore();
      }
      ctx.setLineDash([]);
    }
  }
};
