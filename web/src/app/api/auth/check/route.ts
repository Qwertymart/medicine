// app/api/auth/check/route.ts
import { NextResponse } from "next/server";
import { cookies } from "next/headers";

export async function GET() {
  try {
    // dev
    // if (process.env.FLAG === "development") {
    //   const accessToken = (await cookies()).get("access_token")?.value;

    //   if (!accessToken) {
    //     return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
    //   }

    //   return NextResponse.json(
    //     { authenticated: true, user: { id: 1, username: "mock-user" } },
    //     { status: 200 }
    //   );
    // }

    // Прод
    const accessToken = (await cookies()).get("access_token")?.value;

    if (!accessToken) {
      return NextResponse.json(
        { error: "Access token missing" },
        { status: 401 }
      );
    }

    const response = await fetch(
      `${process.env.DJANGO_API}/api/token/verify/`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${accessToken}`,
        },
        body: JSON.stringify({
          token: accessToken,
        }),
      }
    );

    if (!response.ok) {
      return NextResponse.json({ error: "Invalid token" }, { status: 401 });
    }

    const userData = await response.json();
    return NextResponse.json(userData, { status: 200 });
  } catch {
    return NextResponse.json(
      { error: "Internal Server Error" },
      { status: 500 }
    );
  }
}
