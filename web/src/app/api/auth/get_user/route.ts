/* eslint-disable @typescript-eslint/no-explicit-any */
import { NextResponse } from "next/server";
import { cookies } from "next/headers";

export async function GET() {
  try {
    // if (process.env.FLAG === "development") {
    //   const mockUser: UserInfo = {
    //     firstName: "Вовчик",
    //     lastName: "Бугренков",
    //     username: "qwental_",
    //     email: "vova2005gt@gmail.com",
    //     avatarSrc: "https://i.imgur.com/kEanQzn.jpeg",
    //     photos: [
    //       {
    //         picSrc: "https://i.imgur.com/YzFSzED.jpeg",
    //         date: "27.02.2005",
    //       },
    //       {
    //         picSrc: "https://i.imgur.com/YzFSzED.jpeg",
    //         date: "27.02.2005",
    //       },
    //       {
    //         picSrc: "https://i.imgur.com/YzFSzED.jpeg",
    //         date: "27.02.2005",
    //       },
    //       {
    //         picSrc: "https://i.imgur.com/YzFSzED.jpeg",
    //         date: "27.02.2005",
    //       },
    //       {
    //         picSrc: "https://i.imgur.com/YzFSzED.jpeg",
    //         date: "27.02.2005",
    //       },
    //       {
    //         picSrc: "https://i.imgur.com/YzFSzED.jpeg",
    //         date: "27.02.2005",
    //       },
    //       {
    //         picSrc: "https://i.imgur.com/YzFSzED.jpeg",
    //         date: "27.02.2005",
    //       },
    //       {
    //         picSrc: "https://i.imgur.com/YzFSzED.jpeg",
    //         date: "27.02.2005",
    //       },
    //       {
    //         picSrc: "https://i.imgur.com/YzFSzED.jpeg",
    //         date: "27.02.2005",
    //       },
    //     ],
    //   };
    //   return NextResponse.json(mockUser);
    // }

    const accessToken = (await cookies()).get("access_token")?.value;
    console.log("Access token:", accessToken);

    if (!accessToken) {
      return NextResponse.json(
        { error: "Требуется авторизация" },
        { status: 401 }
      );
    }

    const baseUrl = process.env.DJANGO_API?.replace(/\/$/, "");
    const url = `${baseUrl}/api/users/0/`;

    const response = await fetch(url, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${accessToken}`,
        "Content-Type": "application/json",
      },
    });

    console.log("ответ:", response.status);

    if (!response.ok) {
      return NextResponse.json(
        { error: "Ошибка запроса к Django" },
        { status: 502 }
      );
    }

    const answer = await response.json();
    console.log(":", answer);

    const userData: UserInfo = {
      firstName: answer.first_name || "",
      lastName: answer.last_name || "",
      username: answer.username || "",
      email: answer.email || "",
      avatarSrc: answer.avatar_url || "https://i.imgur.com/kEanQzn.jpeg",
      photos: (answer.photos || []).map((p: any) => ({
        id: p.id,
        image: p.image.startsWith("http") ? p.image : `${baseUrl}${p.image}`,
        created_at: p.created_at,
      })),
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
