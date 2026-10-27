import { Suspense } from "react";

import { LoginPage } from "@/modules/identity";

export default function LoginRoute() {
  return (
    <Suspense>
      <LoginPage />
    </Suspense>
  );
}
