#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

const sceneDir = __dirname;
const rowX = [2, 1.72, 1.44, 1.16, 0.88, 0.6];
const rowZ = [3.26, 2.72, 2.18, 1.64, 1.1, 0.58];
const colY = [1.68, 1.12, 0.56, 0, -0.56, -1.12, -1.68];
const idPattern = /^geo-r(\d{2})-c(\d{2})(?:-|$)/;

function detectNewline(text) {
  return text.includes("\r\n") ? "\r\n" : "\n";
}

function skipString(text, index) {
  index++;
  while (index < text.length) {
    if (text[index] === "\\") {
      index += 2;
      continue;
    }
    if (text[index] === "\"") return index + 1;
    index++;
  }
  throw new Error("Unterminated string");
}

function skipWhitespace(text, index) {
  while (index < text.length && /\s/.test(text[index])) index++;
  return index;
}

function skipValue(text, index) {
  index = skipWhitespace(text, index);
  const open = text[index];
  const close = open === "[" ? "]" : open === "{" ? "}" : null;

  if (close) {
    let depth = 0;
    while (index < text.length) {
      const char = text[index];
      if (char === "\"") {
        index = skipString(text, index);
        continue;
      }
      if (char === open) depth++;
      if (char === close) {
        depth--;
        if (depth === 0) return index + 1;
      }
      index++;
    }
    throw new Error(`Unterminated ${open} value`);
  }

  if (open === "\"") return skipString(text, index);

  while (index < text.length && !/[,\]\}\s]/.test(text[index])) index++;
  return index;
}

function findMatchingBrace(text, start) {
  let depth = 0;
  for (let index = start; index < text.length; index++) {
    const char = text[index];
    if (char === "\"") {
      index = skipString(text, index) - 1;
      continue;
    }
    if (char === "{") depth++;
    if (char === "}") {
      depth--;
      if (depth === 0) return index;
    }
  }
  throw new Error("Unterminated object");
}

function listObjectSpans(text) {
  const objectsKey = text.indexOf("\"objects\"");
  if (objectsKey === -1) return [];

  const arrayStart = text.indexOf("[", objectsKey);
  if (arrayStart === -1) return [];

  const spans = [];
  let depth = 0;
  for (let index = arrayStart; index < text.length; index++) {
    const char = text[index];
    if (char === "\"") {
      index = skipString(text, index) - 1;
      continue;
    }
    if (char === "[") depth++;
    if (char === "]") {
      depth--;
      if (depth === 0) break;
    }
    if (char === "{" && depth === 1) {
      const end = findMatchingBrace(text, index);
      spans.push({ start: index, end: end + 1 });
      index = end;
    }
  }
  return spans;
}

function findTopLevelProperty(objectText, propertyName) {
  let depth = 0;
  for (let index = 0; index < objectText.length; index++) {
    const char = objectText[index];
    if (char === "\"") {
      const stringEnd = skipString(objectText, index);
      if (depth === 1) {
        const raw = objectText.slice(index, stringEnd);
        if (JSON.parse(raw) === propertyName) {
          const colon = skipWhitespace(objectText, stringEnd);
          if (objectText[colon] === ":") {
            const valueStart = skipWhitespace(objectText, colon + 1);
            const valueEnd = skipValue(objectText, valueStart);
            return { keyStart: index, valueStart, valueEnd };
          }
        }
      }
      index = stringEnd - 1;
      continue;
    }
    if (char === "{" || char === "[") depth++;
    if (char === "}" || char === "]") depth--;
  }
  return null;
}

function lineIndentBefore(text, index) {
  const lineStart = text.lastIndexOf("\n", index) + 1;
  const match = text.slice(lineStart, index).match(/^\s*/);
  return match ? match[0] : "";
}

function formatCenter(center, propertyIndent, newline) {
  const valueIndent = `${propertyIndent}  `;
  return [
    "[",
    `${valueIndent}${center[0]},`,
    `${valueIndent}${center[1]},`,
    `${valueIndent}${center[2]}`,
    `${propertyIndent}]`
  ].join(newline);
}

function expectedCenter(id) {
  const match = idPattern.exec(id || "");
  if (!match) return null;

  const row = Number(match[1]);
  const col = Number(match[2]);
  if (row < 1 || row > rowX.length || col < 1 || col > colY.length) {
    throw new Error(`Object id is outside the benchmark matrix: ${id}`);
  }

  return [rowX[row - 1], colY[col - 1], rowZ[row - 1]];
}

function updateObject(objectText, newline) {
  const object = JSON.parse(objectText);
  const center = expectedCenter(object.id);
  if (!center) return { text: objectText, changed: false };

  const centerProperty = findTopLevelProperty(objectText, "center");
  if (centerProperty) {
    const propertyIndent = lineIndentBefore(objectText, centerProperty.keyStart);
    const nextText =
      objectText.slice(0, centerProperty.valueStart) +
      formatCenter(center, propertyIndent, newline) +
      objectText.slice(centerProperty.valueEnd);
    return { text: nextText, changed: nextText !== objectText };
  }

  const shapeProperty = findTopLevelProperty(objectText, "shape");
  if (!shapeProperty) {
    throw new Error(`Object is missing a shape field: ${object.id}`);
  }

  const propertyIndent = lineIndentBefore(objectText, shapeProperty.keyStart);
  let insertAt = shapeProperty.valueEnd;
  insertAt = skipWhitespace(objectText, insertAt);
  if (objectText[insertAt] === ",") insertAt++;
  if (objectText.startsWith("\r\n", insertAt)) insertAt += 2;
  else if (objectText[insertAt] === "\n") insertAt++;

  const inserted =
    `${propertyIndent}"center": ${formatCenter(center, propertyIndent, newline)},${newline}`;
  const nextText = objectText.slice(0, insertAt) + inserted + objectText.slice(insertAt);
  return { text: nextText, changed: true };
}

function updateFile(filePath) {
  const original = fs.readFileSync(filePath, "utf8");
  const newline = detectNewline(original);
  const spans = listObjectSpans(original);

  let changedCount = 0;
  let cursor = 0;
  let updated = "";

  for (const span of spans) {
    updated += original.slice(cursor, span.start);
    const result = updateObject(original.slice(span.start, span.end), newline);
    updated += result.text;
    if (result.changed) changedCount++;
    cursor = span.end;
  }

  updated += original.slice(cursor);
  if (updated !== original) fs.writeFileSync(filePath, updated, "utf8");
  return changedCount;
}

let total = 0;
for (const fileName of fs.readdirSync(sceneDir).sort()) {
  if (!/^geo-r\d{2}\.json$/.test(fileName)) continue;
  const changed = updateFile(path.join(sceneDir, fileName));
  total += changed;
  console.log(`${fileName}: ${changed} center field${changed === 1 ? "" : "s"} updated`);
}
console.log(`Total center fields updated: ${total}`);
