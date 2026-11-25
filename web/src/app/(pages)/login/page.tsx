import { headers } from "next/headers";
import { Suspense } from "react";

import { LoginPage } from "@/modules/identity";

export default async function LoginRoute() {
  let initialLogin;
  if (process.env.NODE_ENV === "development" && process.env.CLASSGAME_LOCAL_LOGIN_PASSWORD) {
    const host = (await headers()).get("host") ?? "";
    if (/^(localhost|127\.0\.0\.1)(:\d+)?$/i.test(host)) {
      initialLogin = {
        email: process.env.CLASSGAME_LOCAL_LOGIN_EMAIL ?? "",
        password: process.env.CLASSGAME_LOCAL_LOGIN_PASSWORD,
      };
    }
  }

  return (
    <Suspense>
      <LoginPage initialLogin={initialLogin} />
    </Suspense>
  );
}
