import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require = createRequire(import.meta.url);

function loadComponent(path, mocks = {}) {
  const code = readFileSync(new URL(path, import.meta.url), "utf8");
  const output = ts.transpileModule(code, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      jsx: ts.JsxEmit.ReactJSX,
      target: ts.ScriptTarget.ES2020,
    },
  }).outputText;

  const mockReact = {
    ...require("react"),
    useState: (initial) => [typeof initial === "function" ? initial() : initial, () => {}],
    useMemo: (fn) => fn(),
    useCallback: (fn) => fn,
    useEffect: () => {},
  };

  const target = { exports: {} };
  vm.runInNewContext(output, {
    exports: target.exports,
    module: target,
    require: (name) => {
      if (name === "react") return mockReact;
      if (name === "react/jsx-runtime") return require("react/jsx-runtime");
      if (name === "lucide-react") return require("lucide-react");
      if (name === "@/lib/i18n") {
        return mocks.i18n ?? {
          useTranslation: () => ({
            t: (key, params) => {
              if (params) {
                let res = key;
                for (const [k, v] of Object.entries(params)) {
                  res = res.replace(`{${k}}`, String(v));
                }
                return res;
              }
              return key;
            },
            direction: "ltr",
          }),
        };
      }
      if (name === "@/features/auth/hooks/use-auth") {
        return {
          useAuth: () => mocks.auth ?? {
            user: { id: "u1", role: "CASHIER", permissions: ["pos:checkout"] },
            hasPermission: (p) => p === "pos:checkout",
            hasRole: (r) => r === "CASHIER",
            isLoading: false,
          },
        };
      }
      if (name === "@/features/auth/hooks/use-capabilities") {
        return {
          useCapabilities: () => mocks.capabilities ?? {
            capabilities: {
              tenant_id: "t1",
              plan: { code: "starter", name: "Starter", status: "ACTIVE" },
              capabilities: {
                "pos.checkout": { allowed: true, reason: "ENTITLED" },
                "pos.daily_summary": { allowed: false, reason: "FEATURE_NOT_ENTITLED" },
              },
              policy_version: 1,
            },
            isLoading: false,
            error: null,
            isServiceUnavailable: false,
            refresh: async () => {},
            checkFeature: (k) => {
              if (k === "pos.checkout") return { allowed: true, reason: "ENTITLED" };
              if (k === "pos.daily_summary") return { allowed: false, reason: "FEATURE_NOT_ENTITLED" };
              return { allowed: false, reason: "FEATURE_NOT_ENTITLED" };
            },
          },
        };
      }
      throw new Error(`Unexpected import in test: ${name}`);
    },
  });

  return target.exports;
}

test("FeatureGate: renders children when tenant is entitled and user has permission", () => {
  const { FeatureGate } = loadComponent("./feature-gate.tsx", {
    auth: {
      user: { id: "u1", role: "CASHIER", permissions: ["pos:checkout"] },
      hasPermission: (p) => p === "pos:checkout",
      hasRole: (r) => r === "CASHIER",
      isLoading: false,
    },
    capabilities: {
      isLoading: false,
      isServiceUnavailable: false,
      checkFeature: (k) => ({ allowed: k === "pos.checkout", reason: "ENTITLED" }),
    },
  });

  const element = FeatureGate({
    featureKey: "pos.checkout",
    requiredPermission: "pos:checkout",
    children: "Protected POS Content",
  });

  // Allowed state returns fragment wrapping children
  assert.equal(element.props.children, "Protected POS Content");
});

test("FeatureGate: renders loading skeleton pulse while evaluating", () => {
  const { FeatureGate } = loadComponent("./feature-gate.tsx", {
    capabilities: {
      isLoading: true,
      isServiceUnavailable: false,
      checkFeature: () => ({ allowed: false, reason: "FEATURE_NOT_ENTITLED" }),
    },
  });

  const element = FeatureGate({
    featureKey: "pos.daily_summary",
    children: "Protected Content",
  });

  assert.match(element.props.className, /animate-pulse/);
  assert.equal(element.props["aria-busy"], "true");
});

test("FeatureGate: renders insufficient_subscription state with upgrade prompt when feature not entitled", () => {
  const { FeatureGate } = loadComponent("./feature-gate.tsx", {
    auth: {
      user: { id: "u1", role: "CASHIER", permissions: ["pos:read"] },
      hasPermission: () => true,
      hasRole: () => true,
      isLoading: false,
    },
    capabilities: {
      isLoading: false,
      isServiceUnavailable: false,
      checkFeature: () => ({ allowed: false, reason: "FEATURE_NOT_ENTITLED" }),
    },
  });

  const element = FeatureGate({
    featureKey: "pos.daily_summary",
    children: "Protected Content",
  });

  assert.match(element.props.className, /border-amber-500\/20/);
  assert.equal(element.props.role, "alert");
});

test("FeatureGate: renders insufficient_permission when user lacks RBAC permission", () => {
  const { FeatureGate } = loadComponent("./feature-gate.tsx", {
    auth: {
      user: { id: "u1", role: "CASHIER", permissions: [] },
      hasPermission: () => false, // No permission
      hasRole: () => false,
      isLoading: false,
    },
    capabilities: {
      isLoading: false,
      isServiceUnavailable: false,
      checkFeature: () => ({ allowed: true, reason: "ENTITLED" }),
    },
  });

  const element = FeatureGate({
    featureKey: "pos.daily_summary",
    requiredPermission: "manager:read",
    children: "Protected Content",
  });

  assert.match(element.props.className, /border-rose-500\/20/);
});

test("FeatureGate: renders owner_disabled when feature is disabled by tenant owner", () => {
  const { FeatureGate } = loadComponent("./feature-gate.tsx", {
    auth: {
      user: { id: "u1", role: "MANAGER", permissions: ["manager:read"] },
      hasPermission: () => true,
      hasRole: () => true,
      isLoading: false,
    },
    capabilities: {
      isLoading: false,
      isServiceUnavailable: false,
      checkFeature: () => ({ allowed: false, reason: "FEATURE_DISABLED" }),
    },
  });

  const element = FeatureGate({
    featureKey: "pos.offline_mode",
    children: "Protected Content",
  });

  assert.match(element.props.className, /border-slate-300/);
});

test("FeatureGate: renders quota_exceeded with usage information when capacity reached", () => {
  const { FeatureGate } = loadComponent("./feature-gate.tsx", {
    auth: {
      user: { id: "u1", role: "MANAGER", permissions: ["manager:read"] },
      hasPermission: () => true,
      hasRole: () => true,
      isLoading: false,
    },
    capabilities: {
      isLoading: false,
      isServiceUnavailable: false,
      checkFeature: () => ({
        allowed: false,
        reason: "QUOTA_EXCEEDED",
        quota_limit: 100,
        current_usage: 100,
      }),
    },
  });

  const element = FeatureGate({
    featureKey: "catalog.max_products",
    children: "Protected Content",
  });

  assert.match(element.props.className, /border-orange-500\/20/);
});

test("FeatureGate: renders service_unavailable with retry button on degraded service", () => {
  const { FeatureGate } = loadComponent("./feature-gate.tsx", {
    auth: {
      user: { id: "u1", role: "MANAGER", permissions: [] },
      hasPermission: () => true,
      hasRole: () => true,
      isLoading: false,
    },
    capabilities: {
      isLoading: false,
      isServiceUnavailable: true,
      checkFeature: () => ({ allowed: false, reason: "SERVICE_UNAVAILABLE" }),
    },
  });

  const element = FeatureGate({
    featureKey: "pos.daily_summary",
    children: "Protected Content",
  });

  assert.equal(element.props.role, "alert");
});

test("FeatureGate: returns null when hideIfDenied is true and feature is not allowed", () => {
  const { FeatureGate } = loadComponent("./feature-gate.tsx", {
    auth: {
      user: { id: "u1", role: "CASHIER", permissions: [] },
      hasPermission: () => true,
      hasRole: () => true,
      isLoading: false,
    },
    capabilities: {
      isLoading: false,
      isServiceUnavailable: false,
      checkFeature: () => ({ allowed: false, reason: "FEATURE_NOT_ENTITLED" }),
    },
  });

  const element = FeatureGate({
    featureKey: "pos.daily_summary",
    hideIfDenied: true,
    children: "Protected Content",
  });

  assert.equal(element, null);
});

test("FeatureGate: executes custom fallback function with accurate GateState parameter", () => {
  const { FeatureGate } = loadComponent("./feature-gate.tsx", {
    auth: {
      user: { id: "u1", role: "CASHIER", permissions: [] },
      hasPermission: () => true,
      hasRole: () => true,
      isLoading: false,
    },
    capabilities: {
      isLoading: false,
      isServiceUnavailable: false,
      checkFeature: () => ({ allowed: false, reason: "FEATURE_NOT_ENTITLED" }),
    },
  });

  let receivedState = null;
  const element = FeatureGate({
    featureKey: "pos.daily_summary",
    fallback: (state) => {
      receivedState = state;
      return `Custom Fallback for ${state}`;
    },
    children: "Protected Content",
  });

  assert.equal(receivedState, "insufficient_subscription");
  assert.equal(element.props.children, "Custom Fallback for insufficient_subscription");
});
