// Generates the PWA icons (192, 512, maskable 512) from the brand SVG mark.
//
// Run: `node scripts/generate-pwa-icons.mjs` (or `npm run generate:icons`).
// Uses @resvg/resvg-js (prebuilt binary, no system deps) so the icons are
// reproducible from the single source of truth in public/favicon.svg.

import { readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { Resvg } from '@resvg/resvg-js';

const here = dirname(fileURLToPath(import.meta.url));
const publicDir = resolve(here, '..', 'public');

const BRAND_BG = '#863bff';
const markSvg = readFileSync(resolve(publicDir, 'favicon.svg'), 'utf8');

/**
 * Wrap the brand mark in a solid brand-color square. `safeZone` (0..1) is the
 * fraction of the canvas kept clear on every side — maskable icons need ~10%
 * so the mark survives the platform's circular/rounded crop.
 */
function composeIcon(size, safeZone) {
  const inner = Math.round(size * (1 - safeZone * 2));
  const offset = Math.round((size - inner) / 2);
  const encoded = Buffer.from(markSvg).toString('base64');
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 ${size} ${size}">
  <rect width="${size}" height="${size}" fill="${BRAND_BG}"/>
  <image x="${offset}" y="${offset}" width="${inner}" height="${inner}" href="data:image/svg+xml;base64,${encoded}"/>
</svg>`;
}

function renderPng(svg, size, filename) {
  const resvg = new Resvg(svg, { fitTo: { mode: 'width', value: size } });
  const png = resvg.render().asPng();
  writeFileSync(resolve(publicDir, filename), png);
  console.log(`wrote public/${filename} (${size}x${size}, ${png.length} bytes)`);
}

// The mark already sits on a transparent field; for the "any" icons we keep a
// small margin, and for the maskable icon a larger safe zone.
renderPng(composeIcon(192, 0.12), 192, 'pwa-192x192.png');
renderPng(composeIcon(512, 0.12), 512, 'pwa-512x512.png');
renderPng(composeIcon(512, 0.2), 512, 'pwa-maskable-512x512.png');
