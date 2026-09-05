/**
 * Tenet Commerce — i18n Type Definitions
 * Type-safe translation schema supporting Indonesian (id), English (en), and Arabic (ar).
 */

export type Locale = "id" | "en" | "ar";
export type Direction = "ltr" | "rtl";

export interface LocaleConfig {
  code: Locale;
  name: string;
  nativeName: string;
  flag: string;
  direction: Direction;
}

export interface TranslationSchema {
  common: {
    actions: {
      cancel: string;
      confirm: string;
      close: string;
      save: string;
      delete: string;
      refresh: string;
      back: string;
      next: string;
      retry: string;
      copy: string;
      copied: string;
      search: string;
      loading: string;
      filter: string;
    };
    status: {
      online: string;
      offline: string;
      completed: string;
      voided: string;
      pending: string;
      failed: string;
      active: string;
    };
    network: {
      connected: string;
      disconnected: string;
    };
    pagination: {
      showing: string;
      total: string;
    };
  };
  nav: {
    pos: string;
    orderHistory: string;
    inventory: string;
    dashboard: string;
    logout: string;
    switchTenant: string;
  };
  auth: {
    title: string;
    subtitle: string;
    badge: string;
    tenantSlug: string;
    tenantSlugPlaceholder: string;
    email: string;
    emailPlaceholder: string;
    password: string;
    passwordPlaceholder: string;
    submit: string;
    submitting: string;
    demoNote: string;
    errorTitle: string;
    invalidCredentials: string;
    tenantNotFound: string;
    validationError: string;
  };
  pos: {
    title: string;
    subtitle: string;
    tabs: {
      register: string;
      history: string;
    };
    catalog: {
      searchPlaceholder: string;
      allCategories: string;
      emptyCatalog: string;
      noMatch: string;
      stockAvailable: string;
      stockLow: string;
      stockEmpty: string;
      halalCertified: string;
      sku: string;
      barcode: string;
      addToCart: string;
    };
    cart: {
      title: string;
      emptyTitle: string;
      emptyDescription: string;
      clear: string;
      subtotal: string;
      tax: string;
      total: string;
      totalItems: string;
      checkout: string;
      stockLimitReached: string;
      viewCart: string;
      backToCatalog: string;
    };
  };
  tender: {
    modalTitle: string;
    modalDescription: string;
    billTotal: string;
    cashReceived: string;
    cashPlaceholder: string;
    presets: {
      exact: string;
    };
    change: string;
    shortage: string;
    confirmCashSale: string;
    processing: string;
    errors: {
      insufficientCash: string;
      idempotencyFailed: string;
    };
  };
  receipt: {
    modalTitle: string;
    modalDescription: string;
    shopHeader: string;
    halalNotice: string;
    trxNumber: string;
    date: string;
    cashier: string;
    itemHeader: string;
    subtotal: string;
    tax: string;
    total: string;
    cash: string;
    change: string;
    barcode: string;
    thankYou: string;
    printAction: string;
    newSaleAction: string;
  };
  history: {
    title: string;
    subtitle: string;
    columns: {
      trxNumber: string;
      time: string;
      totalBill: string;
      method: string;
      status: string;
      action: string;
    };
    emptyHistory: string;
    viewDetail: string;
    voidAction: string;
    detailModal: {
      title: string;
      time: string;
      cashierId: string;
      status: string;
      voidReason: string;
      saleItems: string;
      totalPayment: string;
      close: string;
    };
    voidModal: {
      title: string;
      description: string;
      warningText: string;
      reasonLabel: string;
      reasonPlaceholder: string;
      cancel: string;
      confirmVoid: string;
      processing: string;
      successMessage: string;
      requiredReason: string;
    };
  };
  diagnostic: {
    title: string;
    copyReport: string;
    reportCopied: string;
  };
}
