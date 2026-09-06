import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import vm from "node:vm";
import ts from "typescript";
import * as money from "../../lib/money.ts";

const require = createRequire(import.meta.url);
// Execute production component/event-handler code; no DOM, browser or visual claims.
function load(path, react = require("react"), globals = {}, i18n) {
  const output = ts.transpileModule(readFileSync(new URL(path, import.meta.url), "utf8"), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX, target: ts.ScriptTarget.ES2020 },
  }).outputText;
  const target = { exports: {} };
  vm.runInNewContext(output, {
    ...globals,
    exports: target.exports, module: target,
    require: name => {
      if (name === "react") return react;
      if (name === "react/jsx-runtime" || name === "lucide-react") return require(name);
      if (name === "@/lib/money") return money;
      if (name === "@/lib/utils") return { cn: (...args) => args.filter(Boolean).join(" ") };
      if (name === "@/lib/i18n") return i18n ?? { useTranslation: () => ({ t: (key, values) => `${key} ${values?.name ?? ""}`.trim() }) };
      if (name === "@/components/ui/button") return { Button: "button" };
      if (name === "@/components/ui/badge") return { Badge: "span" };
      if (name === "@/components/ui/modal") return { Modal: "dialog" };
      if (name === "@/components/ui/alert") return { Alert: "aside" };
      if (name === "@/lib/date") return { formatDateTime: value => value };
      throw new Error(`Unexpected import: ${name}`);
    },
  });
  return target.exports;
}
function nodes(tree) {
  if (!tree || typeof tree !== "object") return [];
  if (Array.isArray(tree)) return tree.flatMap(nodes);
  return [tree, ...nodes(tree.props?.children)];
}
const product = { sku: "SKU-12345678901234567890", name: "Long product name ".repeat(10), unit_price: 123456789, stock_quantity: 2 };

test("language selector uses a single named native touch control for every locale", () => {
  const locales = {
    id: { nativeName: "Bahasa Indonesia", direction: "ltr" },
    en: { nativeName: "English", direction: "ltr" },
    ar: { nativeName: "العربية", direction: "rtl" },
  };
  for (const locale of Object.keys(locales)) {
    const changes = [];
    const { LanguageSelector } = load("../../components/ui/language-selector.tsx", require("react"), {}, {
      LOCALES: locales,
      useTranslation: () => ({ locale, setLocale: value => changes.push(value), t: key => key }),
    });
    const tree = LanguageSelector({});
    assert.equal(tree.type, "select");
    assert.equal(tree.props["aria-label"], "nav.language");
    assert.equal(tree.props.value, locale);
    assert.match(tree.props.className, /h-11/);
    assert.match(tree.props.className, /w-20/);
    assert.match(tree.props.className, /min-w-0/);
    const options = nodes(tree).filter(node => node.type === "option");
    assert.equal(options.length, 3);
    for (const option of options) {
      assert.equal(option.props.lang, option.props.value);
      assert.equal(option.props.dir, "ltr");
      assert.equal(option.props.children, option.props.value.toUpperCase());
      assert.equal(option.props["aria-label"], locales[option.props.value].nativeName);
      tree.props.onChange({ target: { value: option.props.value } });
    }
    tree.props.onChange({ target: { value: "unsupported" } });
    assert.deepEqual(changes, ["id", "en", "ar"]);
  }
});

test("language placement keeps login in document flow and header height content-driven", () => {
  // Source layout contract only, not rendered viewport or native-menu verification.
  const login = readFileSync(new URL("../../app/(auth)/login/page.tsx", import.meta.url), "utf8");
  assert.doesNotMatch(login, /absolute top-4 right-4/);
  assert.match(login, /flex justify-end px-4 pt-4/);
  const header = readFileSync(new URL("../../components/layout/header.tsx", import.meta.url), "utf8");
  assert.match(header, /grid-cols-\[minmax\(0,1fr\)_auto\]/);
  assert.doesNotMatch(header, /min-\[480px\]:w-64/);
  assert.doesNotMatch(header, /flex h-14/);
  assert.doesNotMatch(header, /variant="segmented"/);
});

test("product has one native add control, no nested button, full name and disabled guard", () => {
  const { ProductCard } = load("./components/product-card.tsx");
  let adds = 0;
  const render = extra => ProductCard({ product, onAddToCart: () => adds++, ...extra });
  const card = render();
  assert.equal(card.type, "article");
  assert.equal(card.props.onClick, undefined);
  assert.equal(card.props.role, undefined);
  const buttons = nodes(card).filter(node => node.type === "button");
  assert.equal(buttons.length, 1);
  assert.match(buttons[0].props["aria-label"], /Long product name/);
  buttons[0].props.onClick();
  assert.equal(adds, 1);
  for (const extra of [{ disabled: true }, { product: { ...product, stock_quantity: 0 } }]) {
    const button = nodes(render(extra)).find(node => node.type === "button");
    assert.equal(button.props.disabled, true);
    button.props.onClick();
    assert.equal(adds, 1);
  }
  assert.equal(nodes(card).find(node => node.type === "h4").props.children, product.name);
});

test("cart quantity controls identify the product, preserve quantity one, and guard stock limit", () => {
  const { CartPanel } = load("./components/cart-panel.tsx");
  let decrements = 0;
  let removals = 0;
  const render = quantity => CartPanel({
    items: [{ product, quantity, subtotal: quantity * product.unit_price }],
    totals: { totalItems: quantity, subtotal: product.unit_price, tax: 0, total: product.unit_price },
    onUpdateQuantity() {}, onDecrementItem: () => decrements++, onRemoveItem: () => removals++,
    onClearCart() {}, onOpenTender() {},
  });
  const controls = nodes(render(1)).filter(node => node.type === "button");
  const decrease = controls.find(node => node.props["aria-label"]?.startsWith("pos.cart.decreaseProduct"));
  assert.equal(decrease.props.disabled, true);
  decrease.props.onClick();
  assert.equal(decrements, 0);
  assert.match(decrease.props["aria-label"], /Long product name/);
  controls.find(node => node.props["aria-label"]?.startsWith("pos.cart.removeProduct")).props.onClick();
  assert.equal(removals, 1);
  const atLimit = nodes(render(2));
  assert.equal(atLimit.find(node => node.props?.["aria-label"]?.startsWith("pos.cart.increaseProduct")).props.disabled, true);
  atLimit.find(node => node.props?.["aria-label"]?.startsWith("pos.cart.decreaseProduct")).props.onClick();
  assert.equal(decrements, 1);
});

test("production cart updater keeps quantity one; explicit remove still deletes", () => {
  let state = [{ product, quantity: 1, subtotal: product.unit_price }];
  const fakeReact = {
    useState: () => [state, update => { state = update(state); }],
    useCallback: fn => fn,
    useMemo: fn => fn(),
  };
  const { useCart } = load("./hooks/use-cart.ts", fakeReact);
  const cart = useCart();
  cart.decrementItem(product.sku);
  assert.equal(state.length, 1);
  assert.equal(state[0].quantity, 1);
  cart.updateQuantity(product.sku, 2);
  cart.decrementItem(product.sku);
  assert.equal(state[0].quantity, 1);
  assert.equal(state[0].subtotal, product.unit_price);
  cart.removeItem(product.sku);
  assert.equal(state.length, 0);
});

test("modal names its content and guards Escape/backdrop when dismissal is locked", () => {
  const hooks = { useId: () => "test-dialog", useRef: () => ({ current: null }), useEffect() {} };
  const { Modal } = load("../../components/ui/modal.tsx", hooks);
  let closed = 0;
  let prevented = 0;
  const render = dismissible => Modal({ isOpen: true, onClose: () => closed++, title: "Cash", description: "Review", dismissible, children: "Body" });
  const locked = render(false);
  assert.equal(locked.type, "dialog");
  assert.equal(locked.props["aria-labelledby"], "test-dialog-title");
  assert.equal(locked.props["aria-describedby"], "test-dialog-description");
  locked.props.onCancel({ preventDefault: () => prevented++ });
  assert.equal(prevented, 1);
  const target = { getBoundingClientRect: () => ({ left: 10, top: 10, right: 100, bottom: 100 }) };
  locked.props.onClick({ target, currentTarget: target, clientX: 0, clientY: 0 });
  assert.equal(closed, 0);
  assert.equal(nodes(locked).filter(node => node.type === "button").length, 0);
  const open = render(true);
  open.props.onClick({ target, currentTarget: target, clientX: 50, clientY: 50 });
  assert.equal(closed, 0, "clicks inside dialog/scrollbar area do not dismiss");
  open.props.onClick({ target, currentTarget: target, clientX: 0, clientY: 0 });
  assert.equal(closed, 1);
  open.props.onCancel({ preventDefault() {} });
  assert.equal(closed, 2);
});

test("receipt shows confirmed change and actions before long details, without altering receipt", () => {
  let prints = 0;
  const { ReceiptModal } = load("./components/receipt-modal.tsx", require("react"), { window: { print: () => prints++ } });
  const receipt = {
    transaction_number: "TXN-TEST", total_amount: 10000, change_amount: 5000,
    subtotal_amount: 10000, tax_amount: 0, cash_tendered: 15000, created_at: "2026-09-06T00:00:00Z",
    items: Array.from({ length: 20 }, (_, index) => ({ sku: `SKU-${index}`, name: product.name, quantity: 1, unit_price: 500, subtotal: 500 })),
  };
  const before = JSON.stringify(receipt);
  let nextSale = 0;
  const tree = ReceiptModal({ isOpen: true, onClose() {}, receipt, onNewTransaction: () => nextSale++ });
  const all = nodes(tree);
  const changeIndex = all.findIndex(node => node.type === "dt" && node.props.children === "receipt.change");
  const detailsIndex = all.findIndex(node => node.type === "details");
  assert.ok(changeIndex >= 0 && changeIndex < detailsIndex);
  const buttons = all.filter(node => node.type === "button");
  assert.equal(buttons.length, 2);
  assert.ok(all.indexOf(buttons[1]) < detailsIndex);
  buttons[0].props.onClick();
  assert.equal(prints, 1);
  assert.equal(nextSale, 0, "printing does not start a new sale");
  buttons[1].props.onClick();
  assert.equal(nextSale, 1);
  assert.equal(JSON.stringify(receipt), before);
  assert.equal(all.filter(node => node.type === "li").length, 20);
});

test("modal effect opens, resets scroll, focuses heading and restores body overflow on cleanup", () => {
  const body = { style: { overflow: "auto" } };
  const calls = [];
  const dialog = { scrollTop: 99, showModal: () => calls.push("open"), close: () => calls.push("close") };
  const heading = { focus: options => { assert.equal(options.preventScroll, true); calls.push("focus"); } };
  let effect;
  let refIndex = 0;
  const hooks = {
    useId: () => "lifecycle-dialog",
    useRef: () => ({ current: refIndex++ % 2 === 0 ? dialog : heading }),
    useEffect: fn => { effect = fn; },
  };
  const { Modal } = load("../../components/ui/modal.tsx", hooks, { document: { body } });
  const render = isOpen => Modal({ isOpen, onClose() {}, title: "Review", children: "Body" });
  render(false);
  assert.equal(effect(), undefined);
  assert.deepEqual(calls, []);
  render(true);
  const cleanup = effect();
  assert.deepEqual(calls, ["open", "focus"]);
  assert.equal(dialog.scrollTop, 0);
  assert.equal(body.style.overflow, "hidden");
  cleanup();
  assert.equal(body.style.overflow, "auto");
  // Exercise setup/cleanup again as React development effect replay would do.
  dialog.scrollTop = 77;
  const cleanupAgain = effect();
  assert.equal(dialog.scrollTop, 0);
  cleanupAgain();
  assert.deepEqual(calls, ["open", "focus", "close", "open", "focus", "close"]);
  assert.equal(body.style.overflow, "auto");
});

function tender(extra = {}) {
  const hooks = {
    useState: initial => [initial, () => {}], useRef: () => ({ current: false }),
    useEffect() {}, useMemo: fn => fn(),
  };
  const { TenderModal } = load("./components/tender-modal.tsx", hooks);
  return TenderModal({
    isOpen: true, onClose() {}, totalAmount: 10000, cashTendered: 15000,
    onCashTenderedChange() {}, onSubmit() {}, isSubmitting: false,
    errorMessage: null, step: "review", commandReference: "COMMAND-TEST", ...extra,
  });
}

test("tender keeps pending and unknown locked with payment controls unavailable", () => {
  for (const extra of [
    { step: "submitting", isSubmitting: true },
    { step: "unknown_error", errorMessage: "Uncertain result" },
  ]) {
    const tree = tender(extra);
    assert.equal(tree.props.dismissible, false);
    assert.equal(tree.props.hideCloseButton, true);
    const all = nodes(tree);
    assert.equal(all.find(node => node.type === "input").props.disabled, true);
    assert.ok(all.filter(node => node.type === "button").every(node => node.props.disabled));
    if (extra.step === "unknown_error") {
      assert.ok(all.some(node => node.props?.hidden === true));
      assert.ok(all.some(node => node.props?.children === "COMMAND-TEST"));
      assert.match(all.find(node => node.type === "summary").props.className, /min-h-12/);
    }
  }
});

test("tender disables invalid or insufficient cash and submits only through its explicit control", () => {
  for (const amount of [Number.NaN, -1, 1.5, 9999]) {
    const all = nodes(tender({ cashTendered: amount }));
    const buttons = all.filter(node => node.type === "button");
    assert.equal(buttons.at(-1).props.disabled, true);
    assert.equal(all.find(node => node.type === "input").props.autoFocus, undefined);
  }
  let submits = 0;
  const tree = tender({ onSubmit: () => submits++ });
  const all = nodes(tree);
  assert.equal(tree.props.dismissible, true);
  assert.ok(all.some(node => node.props?.children === "tender.presets.label"));
  const submit = all.filter(node => node.type === "button").at(-1);
  assert.equal(submit.props.disabled, false);
  assert.equal(submits, 0);
  submit.props.onClick();
  assert.equal(submits, 1);
});
