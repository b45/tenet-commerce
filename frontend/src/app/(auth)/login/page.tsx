import * as React from "react";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { LoginForm } from "@/features/auth/login-form";
import { LanguageSelector } from "@/components/ui/language-selector";

export default function LoginPage() {
  return (
    <Card className="w-full max-w-[420px] mx-auto border-black/[0.07] shadow-[0_2px_8px_rgba(0,0,0,0.04),0_20px_40px_-8px_rgba(0,0,0,0.06)] rounded-[22px] relative">
      <div className="absolute top-4 right-4 z-10">
        <LanguageSelector variant="compact" />
      </div>
      <CardHeader className="space-y-3 text-center pb-5 pt-8">
        {/* Apple-style Sculpted Monogram */}
        <div className="mx-auto w-14 h-14 rounded-[14px] bg-[#0B0F19] text-white flex items-center justify-center font-bold text-2xl tracking-tighter shadow-[0_4px_12px_rgba(11,15,25,0.25),inset_0_1px_0_rgba(255,255,255,0.2)]">
          TC
        </div>
        <div>
          <h1 className="text-[22px] font-bold tracking-tight text-[#0B0F19]">
            Tenet Commerce
          </h1>
          <p className="text-[13px] text-[#555D6E] mt-0.5 font-medium">
            Retail & Sharia POS Management
          </p>
        </div>
      </CardHeader>
      <CardContent className="px-7 pb-8 pt-1">
        <LoginForm />
      </CardContent>
    </Card>
  );
}
