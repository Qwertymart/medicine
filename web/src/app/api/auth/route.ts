/* eslint-disable @typescript-eslint/no-explicit-any */
import { cookies } from "next/headers";
import { NextResponse } from "next/server";

type AuthAction = "register" | "login" | "refresh" | "logout";

interface MockData {
  register: any;
  login: { access_token: string };
  refresh: { access_token: string };
  logout: object;
}

export async function POST(req: Request) {
  try {
    const { action, ...payload } = await req.json();
    console.log(`[API Auth] Action: ${action}`, payload);

    const isDev = process.env.NODE_ENV === "development";

    // Mock responses for development
    if (isDev) {
      const mockData: MockData = {
        register: { id: 1, ...payload },
        login: { access_token: "mock-access-token" },
        refresh: { access_token: "new-access-token" },
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
      }

      if (["login", "refresh"].includes(action)) {
        const tokens = mockData[action as "login" | "refresh"];
        res.cookies.set("access_token", tokens.access_token, {
          httpOnly: true,
          secure: !isDev,
          sameSite: "lax",
        });
      }

      return res;
    }

    // Production endpoints
    const endpoints: Record<AuthAction, string> = {
      login: "/api/v1/auth/login",
      register: "/api/v1/auth/register",
      refresh: "/api/v1/auth/refresh",
      logout: "/api/v1/auth/logout",
    };

    if (!(action in endpoints)) {
      return NextResponse.json(
        { error: "Invalid action type" },
        { status: 400 }
      );
    }

    const baseUrl = process.env.BACKEND_API?.replace(/\/$/, "");
    const endpoint = endpoints[action as AuthAction];
    const url = `${baseUrl}${endpoint}`;

    // Prepare request body based on action
    let body: any = {};
    
    if (action === "register") {
      // Map to match Go struct: name, last_name, email, password
      body = {
        name: payload.firstName || payload.name,
        last_name: payload.lastName || payload.last_name,
        email: payload.email,
        password: payload.password
      };
    } else if (action === "login") {
      // Map to match Go struct: email, password
      body = {
        email: payload.email,
        password: payload.password
      };
    } else if (action === "refresh") {
      // Refresh token handling
      const refreshToken = (await cookies()).get("refresh_token")?.value;
      body = { refresh_token: refreshToken };
    }

    // Prepare request options
    const options: RequestInit = {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
    };

    if (action !== "logout") {
      options.body = JSON.stringify(body);
    }

    if (action === "logout") {
      const accessToken = (await cookies()).get("access_token")?.value;
      if (accessToken) {
        options.headers = {
          ...options.headers,
          Authorization: `Bearer ${accessToken}`,
        };
      }
    }

    const response = await fetch(url, options);

    if (!response.ok) {
      const errorData = await response.text();
      console.error("Backend error:", errorData);
      return NextResponse.json(
        { error: `Backend error: ${errorData}` },
        { status: response.status }
      );
    }

    const data = await response.json();
    const nextRes = NextResponse.json(data);

    // Handle token setting for login/refresh
    if (["login", "refresh"].includes(action) && data.access_token) {
      nextRes.cookies.set("access_token", data.access_token, {
        httpOnly: true,
        secure: process.env.NODE_ENV === "production",
        sameSite: "lax",
      });
    }

    // Handle logout token removal
    if (action === "logout") {
      nextRes.cookies.delete("access_token");
    }

    return nextRes;
  } catch (error) {
    console.error("Global error handler:", error);
    return NextResponse.json(
      { error: "Internal Server Error" },
      { status: 500 }
    );
  }
}
