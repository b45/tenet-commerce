import * as React from "react";
import { Card, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ShoppingCart } from "lucide-react";

export default function POSPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-[var(--color-text-primary)]">
            Terminal Kasir POS
          </h1>
          <p className="text-sm text-[var(--color-text-secondary)]">
            Pencarian produk, pemindaian barcode, dan pemrosesan pembayaran tunai.
          </p>
        </div>
        <Badge variant="success" className="text-xs">
          Online
        </Badge>
      </div>

      <Card className="border-dashed border-2">
        <CardHeader className="text-center py-12">
          <div className="mx-auto w-12 h-12 rounded-full bg-indigo-50 text-[var(--color-action-primary)] flex items-center justify-center mb-3">
            <ShoppingCart className="w-6 h-6" />
          </div>
          <CardTitle className="text-lg">Workspace Kasir POS (Step 2)</CardTitle>
          <CardDescription className="max-w-md mx-auto mt-2">
            Antarmuka penuh keranjang belanja, integrasi katalog online, dan dialog pembayaran tunai dengan tombol uang pas akan diimplementasikan pada paket Step 2.
          </CardDescription>
        </CardHeader>
      </Card>
    </div>
  );
}
