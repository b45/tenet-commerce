import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import vm from "node:vm";
import ts from "typescript";
import * as money from "../../lib/money.ts";

const require = createRequire(import.meta.url);
// Execute production component/event-handler code; no DOM, browser or visual claims.
function load(path, react = require("react")) {
  const output = ts.transpileModule(readFileSync(new URL(path, import.meta.url), "utf8"), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX, target: ts.ScriptTarget.ES2020 },
  }).outputText;
  const target = { exports: {} };
  vm.runInNewContext(output, {
    exports: target.exports, module: target,
    require: name => {
      if (name === "react") return react;
      if (name === "react/jsx-runtime" || name === "lucide-react") return require(name);
      if (name === "@/lib/money") return money;
      if (name === "@/lib/utils") return { cn: (...args) => args.filter(Boolean).join(" ") };
      if (name === "@/lib/i18n") return { useTranslation: () => ({ t: (key, values) => `${key} ${values?.name ?? ""}`.trim() }) };
      if (name === "@/components/ui/button") return { Button: "button" };
      if (name === "@/components/ui/badge") return { Badge: "span" };
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
