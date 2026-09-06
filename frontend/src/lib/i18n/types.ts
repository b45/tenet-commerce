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
    brand: string;
    openMenu: string;
    closeMenu: string;
    navigation: string;
    unknownTenant: string;
    language: string;
    sessionLoading: string;
    certificates: string;
    procurement: string;
    ledger: string;
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
      generalCategory: string;
      addProduct: string;
      title: string;
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
      removeProduct: string;
      decreaseProduct: string;
      increaseProduct: string;
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
    unknownTitle: string;
    rejectedTitle: string;
    unknownHelp: string;
    referenceTitle: string;
    referenceHelp: string;
    invalidAmount: string;
    modalTitle: string;
    modalDescription: string;
    billTotal: string;
    cashReceived: string;
    cashPlaceholder: string;
    presets: {
      label: string;
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
  inventory: {
    title: string;
    subtitle: string;
    searchPlaceholder: string;
    filterAll: string;
    filterLowStock: string;
    filterOutOfStock: string;
    filterCategory: string;
    allCategories: string;
    addProduct: string;
    adjustStock: string;
    refresh: string;
    unit: string;
    close: string;
    lowStockBanner: {
      alertTitle: string;
      alertMessage: string;
      viewItems: string;
      dismiss: string;
    };
    table: {
      sku: string;
      productName: string;
      category: string;
      costPrice: string;
      unitPrice: string;
      stock: string;
      status: string;
      actions: string;
      active: string;
      inactive: string;
      halalBadge: string;
      halalNotice: string;
      emptyTitle: string;
      emptyDescription: string;
      editAction: string;
      adjustAction: string;
      deleteAction: string;
      loading: string;
      generalCategory: string;
    };
    productModal: {
      createTitle: string;
      editTitle: string;
      sku: string;
      skuPlaceholder: string;
      barcode: string;
      barcodePlaceholder: string;
      name: string;
      namePlaceholder: string;
      category: string;
      selectCategory: string;
      unitPrice: string;
      costPrice: string;
      initialStock: string;
      reorderThreshold: string;
      description: string;
      descriptionPlaceholder: string;
      halalCertified: string;
      halalCertifiedNotice: string;
      activeStatus: string;
      save: string;
      saving: string;
      cancel: string;
      closeModal: string;
      createSuccess: string;
      updateSuccess: string;
      errors: {
        nameRequired: string;
        skuRequired: string;
        createFailed: string;
        updateFailed: string;
      };
    };
    adjustModal: {
      title: string;
      description: string;
      currentStock: string;
      adjustType: string;
      types: {
        add: string;
        subtract: string;
        set: string;
      };
      quantity: string;
      reason: string;
      reasons: {
        damage: string;
        expired: string;
        auditCorrection: string;
        restock: string;
        other: string;
      };
      notes: string;
      notesPlaceholder: string;
      previewDelta: string;
      newStockPreview: string;
      submit: string;
      submitting: string;
      cancel: string;
      closeModal: string;
      successMessage: string;
      errors: {
        quantityPositive: string;
        adjustFailed: string;
      };
    };
    deleteDialog: {
      title: string;
      message: string;
      confirm: string;
      cancel: string;
      deleting: string;
      closeDialog: string;
      successMessage: string;
      errors: {
        deleteFailed: string;
      };
    };
    permissions: {
      readOnlyTooltip: string;
      writeRequired: string;
    };
  };
  entitlements: {
    loading: string;
    insufficientSubscription: {
      title: string;
      description: string;
      upgradeAction: string;
    };
    insufficientPermission: {
      title: string;
      description: string;
    };
    disabledByOwner: {
      title: string;
      description: string;
    };
    quotaExceeded: {
      title: string;
      description: string;
      usageNotice: string;
    };
    serviceUnavailable: {
      title: string;
      description: string;
      retryAction: string;
    };
    tierBadge: {
      starter: string;
      growth: string;
      enterprise: string;
    };
  };
  dailySummary: {
    buttonLabel: string;
    modalTitle: string;
    modalDescription: string;
    dateLabel: string;
    filterToday: string;
    metrics: {
      grossSales: string;
      netSales: string;
      discounts: string;
      cogs: string;
      grossProfit: string;
      margin: string;
    };
    orders: {
      title: string;
      total: string;
      completed: string;
      voided: string;
    };
    payments: {
      title: string;
      method: string;
      count: string;
      amount: string;
      cash: string;
      qris: string;
    };
    actions: {
      print: string;
      close: string;
      refresh: string;
    };
    emptyState: string;
    loading: string;
    thermalHeader: string;
    thermalFooter: string;
    printTime: string;
    endOfReport: string;
  };
  supplyChain: {
    title: string;
    subtitle: string;
    badge: string;
    addSupplier: string;
    filterAll: string;
    filterValid: string;
    filterExpiringSoon: string;
    filterExpired: string;
    filterNoCert: string;
    searchPlaceholder: string;
    loading: string;
    emptyState: string;
    stats: {
      totalSuppliers: string;
      validCertificates: string;
      expiringSoon: string;
      expiredBlocked: string;
    };
    alertBanner: {
      title: string;
      description: string;
      viewExpiring: string;
    };
    table: {
      supplier: string;
      contact: string;
      certificate: string;
      authority: string;
      validUntil: string;
      status: string;
      actions: string;
      noCertificate: string;
      viewHistory: string;
      renewCert: string;
    };
    status: {
      valid: string;
      expiringSoon: string;
      expired: string;
      revoked: string;
      none: string;
    };
    certTypes: {
      allTypes: string;
      halal: string;
      bpom: string;
      regalkes: string;
      michelin: string;
      isoHaccp: string;
      other: string;
    };
    modal: {
      createSupplierTitle: string;
      createSupplierDesc: string;
      renewCertTitle: string;
      renewCertDesc: string;
      historyTitle: string;
      historyDesc: string;
      codeLabel: string;
      nameLabel: string;
      contactPersonLabel: string;
      emailLabel: string;
      phoneLabel: string;
      hasInitialCert: string;
      certTypeLabel: string;
      customCertTypePlaceholder: string;
      certNumberLabel: string;
      authorityLabel: string;
      scopeLabel: string;
      validFromLabel: string;
      expiryDateLabel: string;
      docUrlLabel: string;
      submitSave: string;
      submitRenew: string;
      cancel: string;
      saving: string;
      revokeAction: string;
      revokeConfirm: string;
      revoking: string;
      noHistory: string;
      successCreated: string;
      successRenewed: string;
      successRevoked: string;
    };
    unauthorizedTitle: string;
    unauthorizedDesc: string;
  };
}
