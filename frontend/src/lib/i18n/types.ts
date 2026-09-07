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
    offline: {
      onlineStatus: string;
      offlineStatus: string;
      cachedNotice: string;
      lastSync: string;
      syncing: string;
      restoredCart: string;
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
    overviewCard: {
      title: string;
      totalSkus: string;
      totalUnits: string;
      lowStock: string;
      outOfStock: string;
      location: string;
      asOf: string;
    };
    procurementDraft: {
      action: string;
      title: string;
      description: string;
      supplierLabel: string;
      productLabel: string;
      thresholdNotice: string;
      suggestedQty: string;
      createPOButton: string;
      close: string;
      successMessage: string;
    };
    lowStockBanner: {
      alertTitle: string;
      alertMessage: string;
      viewItems: string;
      dismiss: string;
      reorderAction: string;
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
      stockCardAction: string;
      procureAction: string;
      deleteAction: string;
      loading: string;
      generalCategory: string;
    };
    stockCardModal: {
      title: string;
      subtitle: string;
      openingBalance: string;
      closingBalance: string;
      currentStock: string;
      totalMovements: string;
      filterDate: string;
      filterAllDates: string;
      startDate: string;
      endDate: string;
      applyFilter: string;
      resetFilter: string;
      columns: {
        dateTime: string;
        type: string;
        sourceDoc: string;
        quantityDelta: string;
        runningBalance: string;
        actorOrReason: string;
      };
      movementTypes: {
        opening: string;
        inbound: string;
        outbound: string;
        adjustment: string;
      };
      sourceTypes: {
        purchaseOrder: string;
        goodsReceipt: string;
        posTransaction: string;
        stockOpname: string;
        manual: string;
      };
      emptyMovements: string;
      loading: string;
      errorFetch: string;
      close: string;
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
  dashboard: {
    title: string;
    subtitle: string;
    viewFilter: {
      today: string;
      allTime: string;
    };
    metrics: {
      grossSales: string;
      netSales: string;
      todayOrders: string;
      avgOrderValue: string;
      activeCatalog: string;
      lowStockAlerts: string;
      halalCompliance: string;
      ledgerIntegrity: string;
    };
    units: {
      transactions: string;
      skus: string;
      alerts: string;
      activeCertificates: string;
      expiringCertificates: string;
      balanced: string;
      unbalanced: string;
      journalEntriesToday: string;
    };
    sections: {
      lowStockTitle: string;
      lowStockDesc: string;
      complianceTitle: string;
      complianceDesc: string;
      operationalTitle: string;
      operationalDesc: string;
    };
    alerts: {
      noLowStock: string;
      noExpiringCerts: string;
      daysRemaining: string;
      expired: string;
      expiringSoon: string;
      currentStock: string;
      threshold: string;
      viewInventoryAction: string;
      viewSuppliersAction: string;
    };
    emptyTenant: string;
    unavailable: string;
    loading: string;
    errorTitle: string;
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
      createPO: string;
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
    receiving: {
      title: string;
      remaining: string;
      deliveredQty: string;
      acceptedQty: string;
      rejectedQty: string;
      rejectionReason: string;
      rejectionReasonPlaceholder: string;
      notes: string;
      notesPlaceholder: string;
      cancel: string;
      submit: string;
      qcNotice: string;
      successTitle: string;
      grNumber: string;
      done: string;
      errors: {
        deliveredPositive: string;
        arithmeticMismatch: string;
        exceedsRemaining: string;
        qcReasonRequired: string;
        failed: string;
      };
    };
  };
  ledger: {
    title: string;
    subtitle: string;
    tabs: {
      entries: string;
      accounts: string;
      trialBalance: string;
    };
    entries: {
      title: string;
      entryNumber: string;
      date: string;
      sourceDoc: string;
      memo: string;
      status: string;
      debit: string;
      credit: string;
      actions: string;
      viewDetail: string;
      emptyState: string;
      balanced: string;
      unbalanced: string;
      postedStatus: string;
      reversedStatus: string;
    };
    detail: {
      title: string;
      entryInfo: string;
      linesTitle: string;
      accountCode: string;
      accountName: string;
      debit: string;
      credit: string;
      total: string;
      sourceDocument: string;
      viewSource: string;
      close: string;
      reversedNotice: string;
    };
    accounts: {
      title: string;
      code: string;
      name: string;
      type: string;
      zakatEligible: string;
      status: string;
      activeStatus: string;
      inactiveStatus: string;
      emptyState: string;
    };
    trialBalance: {
      title: string;
      asOfDate: string;
      accountCode: string;
      accountName: string;
      accountType: string;
      totalDebit: string;
      totalCredit: string;
      netBalance: string;
      total: string;
      isBalanced: string;
      needsReconciliation: string;
      emptyState: string;
    };
    unauthorizedTitle: string;
    unauthorizedDesc: string;
  };
}

