import fs from "node:fs";
import path from "node:path";
import vm from "node:vm";
import { fileURLToPath } from "node:url";

const directory = path.dirname(fileURLToPath(import.meta.url));
const redesignRoot = path.resolve(directory, "..");
const manifest = JSON.parse(fs.readFileSync(path.join(redesignRoot, "screen-manifest.json"), "utf8"));
const fixture = JSON.parse(fs.readFileSync(path.join(redesignRoot, "fixture-state.json"), "utf8"));
const context = { window: {} };
vm.runInNewContext(fs.readFileSync(path.join(directory, "prototype-data.js"), "utf8"), context);
const prototype = context.window.NESTRR_PROTOTYPE;
const ids = new Set(manifest.screens.map((screen) => screen.id));
const errors = [];

for (const screen of manifest.screens) {
  const asset = path.join(redesignRoot, screen.file);
  if (!fs.existsSync(asset)) errors.push(`${screen.id}: falta ${screen.file}`);
  const hotspots = prototype.hotspots[screen.id];
  if (!hotspots?.length) errors.push(`${screen.id}: no tiene hotspots`);
  for (const hotspot of hotspots || []) {
    if (!ids.has(hotspot.target)) errors.push(`${screen.id}: destino inválido ${hotspot.target}`);
    for (const key of ["x", "y", "w", "h"]) {
      if (!Number.isFinite(hotspot[key])) errors.push(`${screen.id}/${hotspot.action}: ${key} inválido`);
    }
    if (hotspot.x < 0 || hotspot.y < 0 || hotspot.x + hotspot.w > 100.1 || hotspot.y + hotspot.h > 100.1) {
      errors.push(`${screen.id}/${hotspot.action}: región fuera de la captura`);
    }
  }
}

for (const [flowId, flow] of Object.entries(prototype.flows)) {
  if (flow.steps.length < 2) errors.push(`${flowId}: recorrido demasiado corto`);
  for (const target of flow.steps) {
    if (!ids.has(target)) errors.push(`${flowId}: pantalla inválida ${target}`);
  }
}

const show = fixture.titles["threshold-harbor"];
if (show.availability.available !== 5 || show.availability.total !== 8) errors.push("fixture: Puerto Umbral debe permanecer 5/8 disponible");
if (show.watched.watched !== 3 || show.watched.total !== 8) errors.push("fixture: Puerto Umbral debe permanecer 3/8 visto");
if (show.episodes.S01E03 !== "remote") errors.push("fixture: S01E03 debe ser el episodio ausente");
if (show.autoplay.from !== "S01E05" || show.autoplay.to !== "S01E06") errors.push("fixture: autoplay debe ser S01E05 → S01E06");
if (!fixture.titles["blind-orbit"].path.includes("Blind Orbit")) errors.push("fixture: la ruta física debe usar el título inglés");
if (fixture.serviceRules["plarr-down"].play || fixture.serviceRules["plarr-down"].download) errors.push("fixture: Plarr caído debe ocultar Play/Download");

if (errors.length) {
  console.error(errors.join("\n"));
  process.exit(1);
}

const hotspotCount = Object.values(prototype.hotspots).reduce((sum, hotspots) => sum + hotspots.length, 0);
console.log(`OK: ${manifest.screens.length} pantallas, ${hotspotCount} hotspots, ${Object.keys(prototype.flows).length} recorridos.`);
