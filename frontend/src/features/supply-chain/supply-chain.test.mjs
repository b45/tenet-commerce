import test from "node:test";
import assert from "node:assert/strict";

/**
 * Unit tests for Halal Supply Chain & Supplier Certification domain logic
 */

test("SupplyChain: correctly identifies expired vs expiring soon vs valid certificates", () => {
  const now = new Date();

  // Helper to compute status matching backend logic
  function computeStatus(expiryDateStr) {
    if (!expiryDateStr) return "NONE";
    const expiry = new Date(expiryDateStr);
    const diffDays = Math.ceil((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
    if (diffDays < 0) return "EXPIRED";
    if (diffDays <= 30) return "EXPIRING_SOON";
    return "VALID";
  }

  // 1. Expired (in the past)
  const pastDate = new Date(now.getTime() - 5 * 24 * 60 * 60 * 1000).toISOString();
  assert.equal(computeStatus(pastDate), "EXPIRED");

  // 2. Expiring soon (< 30 days)
  const soonDate = new Date(now.getTime() + 15 * 24 * 60 * 60 * 1000).toISOString();
  assert.equal(computeStatus(soonDate), "EXPIRING_SOON");

  // 3. Valid (> 30 days)
  const validDate = new Date(now.getTime() + 180 * 24 * 60 * 60 * 1000).toISOString();
  assert.equal(computeStatus(validDate), "VALID");
});

test("SupplyChain: aggregates expiry summary counts accurately", () => {
  const suppliers = [
    { id: "1", company_name: "Supplier A", compliance_certificate: { computed_status: "VALID" } },
    { id: "2", company_name: "Supplier B", compliance_certificate: { computed_status: "VALID" } },
    { id: "3", company_name: "Supplier C", compliance_certificate: { computed_status: "EXPIRING_SOON" } },
    { id: "4", company_name: "Supplier D", compliance_certificate: { computed_status: "EXPIRED" } },
    { id: "5", company_name: "Supplier E", compliance_certificate: { computed_status: "REVOKED" } },
    { id: "6", company_name: "Supplier F", compliance_certificate: null },
  ];

  let valid = 0;
  let expiringSoon = 0;
  let expiredOrRevoked = 0;
  let missing = 0;

  for (const s of suppliers) {
    if (!s.compliance_certificate) {
      missing++;
    } else if (s.compliance_certificate.computed_status === "VALID") {
      valid++;
    } else if (s.compliance_certificate.computed_status === "EXPIRING_SOON") {
      expiringSoon++;
    } else if (
      s.compliance_certificate.computed_status === "EXPIRED" ||
      s.compliance_certificate.computed_status === "REVOKED"
    ) {
      expiredOrRevoked++;
    }
  }

  assert.equal(valid, 2);
  assert.equal(expiringSoon, 1);
  assert.equal(expiredOrRevoked, 2);
  assert.equal(missing, 1);
  assert.equal(suppliers.length, 6);
});

test("SupplyChain: filters suppliers correctly based on search query and status filter", () => {
  const suppliers = [
    {
      id: "1",
      code: "SUPP-001",
      company_name: "PT Berkah Abadi",
      compliance_certificate: { certificate_number: "ID001122", issuing_authority: "BPJPH", computed_status: "VALID" },
    },
    {
      id: "2",
      code: "SUPP-002",
      company_name: "CV Mandiri Sejahtera",
      compliance_certificate: { certificate_number: "ID003344", issuing_authority: "MUI", computed_status: "EXPIRING_SOON" },
    },
    {
      id: "3",
      code: "SUPP-003",
      company_name: "PT Halal Nusantara",
      compliance_certificate: { certificate_number: "ID005566", issuing_authority: "BPJPH", computed_status: "EXPIRED" },
    },
  ];

  // Search by code
  const matchCode = suppliers.filter((s) => s.code.toLowerCase().includes("supp-001"));
  assert.equal(matchCode.length, 1);
  assert.equal(matchCode[0].id, "1");

  // Search by company name
  const matchName = suppliers.filter((s) => s.company_name.toLowerCase().includes("nusantara"));
  assert.equal(matchName.length, 1);
  assert.equal(matchName[0].id, "3");

  // Filter by EXPIRING_SOON
  const matchStatus = suppliers.filter((s) => s.compliance_certificate?.computed_status === "EXPIRING_SOON");
  assert.equal(matchStatus.length, 1);
  assert.equal(matchStatus[0].id, "2");
});

test("SupplyChain: filters suppliers correctly based on standard cert_type presets", () => {
  const suppliers = [
    {
      id: "1",
      company_name: "PT Halal Pangan",
      compliance_certificate: { cert_type: "HALAL", certificate_number: "ID1122" },
    },
    {
      id: "2",
      company_name: "PT Bio Farma Alkes",
      compliance_certificate: { cert_type: "REGALKES", certificate_number: "AKD1002" },
    },
    {
      id: "3",
      company_name: "PT Sari Nutrisi",
      compliance_certificate: { cert_type: "BPOM", certificate_number: "MD2233" },
    },
    {
      id: "4",
      company_name: "Le Petit Gourmet",
      compliance_certificate: { cert_type: "MICHELIN", certificate_number: "MICH-3STAR" },
    },
    {
      id: "5",
      company_name: "PT Agro Rainforest",
      compliance_certificate: { cert_type: "RAINFOREST", certificate_number: "RF-9988" },
    },
    {
      id: "6",
      company_name: "CV Toko Umum",
      compliance_certificate: null,
    },
  ];

  // Helper matching useSuppliers certTypeFilter logic
  function filterByType(list, typeFilter) {
    if (typeFilter === "ALL") return list;
    return list.filter((s) => {
      if (!s.compliance_certificate) return false;
      const t = s.compliance_certificate.cert_type?.toUpperCase() || "";
      if (typeFilter === "OTHER") {
        const presets = ["HALAL", "BPOM", "REGALKES", "MICHELIN", "ISO_HACCP"];
        return !presets.includes(t);
      }
      return t === typeFilter;
    });
  }

  // Filter BPOM
  const bpomMatches = filterByType(suppliers, "BPOM");
  assert.equal(bpomMatches.length, 1);
  assert.equal(bpomMatches[0].id, "3");

  // Filter REGALKES
  const regalkesMatches = filterByType(suppliers, "REGALKES");
  assert.equal(regalkesMatches.length, 1);
  assert.equal(regalkesMatches[0].id, "2");

  // Filter MICHELIN
  const michelinMatches = filterByType(suppliers, "MICHELIN");
  assert.equal(michelinMatches.length, 1);
  assert.equal(michelinMatches[0].id, "4");

  // Filter OTHER (custom standard, e.g. RAINFOREST)
  const otherMatches = filterByType(suppliers, "OTHER");
  assert.equal(otherMatches.length, 1);
  assert.equal(otherMatches[0].id, "5");
});

