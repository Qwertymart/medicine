import { NextResponse } from "next/server";
import { cookies } from "next/headers";

interface UserInfo {
  firstName: string;
  lastName: string;
  username: string;
  email: string;
  avatarSrc: string;
  photos: Array<{
    id: number;
    image: string;
    created_at: string;
  }>;
}

export async function GET() {
  try {
    const accessToken = (await cookies()).get("access_token")?.value;

    if (!accessToken) {
      return NextResponse.json(
        { error: "Требуется авторизация" },
        { status: 401 }
      );
    }

    const baseUrl = process.env.BACKEND_API;
    const response = await fetch(`${baseUrl}/api/v1/auth/me`, {
      headers: {
        Authorization: `Bearer ${accessToken}`,
      },
      credentials: "include",
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: "Ошибка запроса к серверу" },
        { status: 502 }
      );
    }

    const data = await response.json();
    const user = data.user;

    const userData: UserInfo = {
      firstName: user.name || "",
      lastName: user.last_name || "",
      username: user.username || "",
      email: user.email || "",
      avatarSrc: user.avatar_url || "https://i.imgur.com/kEanQzn.jpeg",
      photos: user.photos || [],
    };

    return NextResponse.json(userData);
  } catch (error) {
    console.error("Server error:", error);
    return NextResponse.json(
      { error: "Внутренняя ошибка сервера" },
      { status: 500 }
    );
  }
}
