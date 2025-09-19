/* eslint-disable @typescript-eslint/no-explicit-any */
import { cookies } from "next/headers";
import { NextResponse } from "next/server";

type AuthAction = "register" | "login" | "refresh" | "logout";

interface MockData {
  register: any;
  login: { access: string; refresh: string };
  refresh: { access: string; refresh: string };
  logout: object;
}

export async function POST(req: Request) {
  try {
    const { action, ...payload } = await req.json();

    console.log(`[API Auth] Action: ${action}`, payload);

    const isDev = process.env.FLAG === "development";

    if (isDev) {
      const mockData: MockData = {
        register: { id: 1, ...payload },
        login: { access: "mock-access", refresh: "mock-refresh" },
        refresh: { access: "new-access", refresh: "new-refresh" },
        logout: {},
      };

      if (!(action in mockData)) {
        return NextResponse.json(
          { error: "Invalid action type" },
          { status: 400 }
        );
      }

      const res = NextResponse.json(mockData[action as AuthAction]);

      if (action === "logout") {
        res.cookies.delete("access_token");
        res.cookies.delete("refresh_token");
      }

      if (["login", "refresh"].includes(action)) {
        const tokens = mockData[action as "login" | "refresh"];
        res.cookies.set("access_token", tokens.access);
        res.cookies.set("refresh_token", tokens.refresh);
      }

      return res;
    }

    const endpoints: Record<AuthAction, string> = {
      login: "/api/token/",
      register: "/api/auth/register/",
      refresh: "/api/token/refresh/",
      logout: "/api/auth/logout/",
    };

    if (!(action in endpoints)) {
      return NextResponse.json(
        { error: "Invalid action type" },
        { status: 400 }
      );
    }

    const baseUrl = process.env.DJANGO_API?.replace(/\/$/, "");
    const endpoint = endpoints[action as AuthAction].replace(/^\//, "");
    const url = `${baseUrl}/${endpoint}`;

    console.log("url = ", url);

    let body = payload;
    console.log("body = ", JSON.stringify(body));

if (action === "logout") {
  const refreshToken = (await cookies()).get("refresh_token")?.value;
  const accessToken = (await cookies()).get("access_token")?.value;

  if (!refreshToken) {
    return NextResponse.json(
      { error: "No refresh token found" },
      { status: 401 }
    );
  }

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });

  if (!response.ok) {
    const errorData = await response.text();
    console.error("Backend error:", {
      status: response.status,
      url,
      errorData,
    });

    return NextResponse.json(
      { error: `Backend error: ${errorData}` },
      { status: response.status }
    );
  }

  const nextRes = NextResponse.json({ success: true });
  nextRes.cookies.delete("access_token");
  nextRes.cookies.delete("refresh_token");
  return nextRes;
}


    if (action === "logout" || action === "refresh") {
      body = { refresh: (await cookies()).get("refresh_token")?.value };
    }

    let response;
    try {
      response = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
    } catch (error) {
      console.error("Network error:", error);
      return NextResponse.json(
        { error: `Network error: ${(error as Error).message}` },
        { status: 500 }
      );
    }

    if (action === "logout"){
      
    }

    if (!response.ok) {
      const errorData = await response.text();
      console.error("Backend error:", {
        status: response.status,
        url,
        errorData,
      });

      return NextResponse.json(
        { error: `Backend error: ${errorData}` },
        { status: response.status }
      );
    }

    const data = await response.json();
    const nextRes = NextResponse.json(data);

    if (["login", "refresh"].includes(action)) {
      if (data.access) {
        nextRes.cookies.set("access_token", data.access, {
          httpOnly: true,
          secure: !isDev,
          sameSite: "strict",
          path: "/",
        });
      }
      if (data.refresh) {
        nextRes.cookies.set("refresh_token", data.refresh, {
          httpOnly: true,
          secure: !isDev,
          sameSite: "strict",
          path: "/",
        });
      }
    }

    return nextRes;
  } catch (error) {
    console.error("Global error handler:", error);
    return NextResponse.json(
      {
        error: "Internal Server Error",
        details: error instanceof Error ? error.message : "Unknown error",
      },
      { status: 500 }
    );
  }
}
