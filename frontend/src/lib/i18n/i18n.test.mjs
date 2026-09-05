import test from "node:test";
import assert from "node:assert/strict";
import { id } from "./locales/id.ts";
import { en } from "./locales/en.ts";
import { ar } from "./locales/ar.ts";

const DICTIONARIES = { id, en, ar };

function translateTest(locale, path, params) {
  const dict = DICTIONARIES[locale] || DICTIONARIES.id;
  const keys = path.split(".");

  let current = dict;
  for (const k of keys) {
    if (current && typeof current === "object" && k in current) {
      current = current[k];
    } else {
      current = undefined;
      break;
    }
  }

  if (typeof current !== "string") {
    let fallback = DICTIONARIES.id;
    for (const k of keys) {
      if (fallback && typeof fallback === "object" && k in fallback) {
        fallback = fallback[k];
      } else {
        fallback = undefined;
        break;
      }
    }
    current = typeof fallback === "string" ? fallback : path;
  }

  let text = String(current);
  if (params) {
    for (const [key, value] of Object.entries(params)) {
      text = text.replace(new RegExp(`\\{${key}\\}`, "g"), String(value));
    }
  }

  return text;
}

// Recursive key flattening helper
function getLeafKeys(obj, prefix = "") {
  let keys = [];
  for (const [key, value] of Object.entries(obj)) {
    const fullPath = prefix ? `${prefix}.${key}` : key;
    if (value && typeof value === "object" && !Array.isArray(value)) {
      keys = keys.concat(getLeafKeys(value, fullPath));
    } else {
      keys.push(fullPath);
    }
  }
  return keys.sort();
}

test("i18n: dictionary key parity across id, en, and ar", () => {
  const idKeys = getLeafKeys(id);
  const enKeys = getLeafKeys(en);
  const arKeys = getLeafKeys(ar);

  assert.ok(idKeys.length > 50, "Indonesian dictionary should have comprehensive coverage");
  assert.equal(enKeys.length, idKeys.length, "English dictionary must have exact 1:1 key parity with Indonesian");
  assert.equal(arKeys.length, idKeys.length, "Arabic dictionary must have exact 1:1 key parity with Indonesian");

  assert.deepEqual(enKeys, idKeys, "Every key path in Indonesian must exist in English");
  assert.deepEqual(arKeys, idKeys, "Every key path in Indonesian must exist in Arabic");
});

test("i18n: translation resolution and variable interpolation", () => {
  const textId = translateTest("id", "history.voidModal.warningText", { number: "TXN-001", amount: "Rp 50.000" });
  assert.equal(textId, "Anda akan membatalkan transaksi TXN-001 senilai Rp 50.000.");

  const textEn = translateTest("en", "history.voidModal.warningText", { number: "TXN-001", amount: "Rp 50.000" });
  assert.equal(textEn, "You are voiding transaction TXN-001 valued at Rp 50.000.");

  const textAr = translateTest("ar", "history.voidModal.warningText", { number: "TXN-001", amount: "Rp 50.000" });
  assert.equal(textAr, "أنت على وشك إلغاء الفاتورة TXN-001 بقيمة Rp 50.000.");
});

test("i18n: fallback to Indonesian on missing key or non-existent path", () => {
  const fallback = translateTest("en", "non.existent.path");
  assert.equal(fallback, "non.existent.path", "Missing paths should safely return the path string without crashing");
});
