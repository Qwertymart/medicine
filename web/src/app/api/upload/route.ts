import { cookies } from "next/headers";
import { NextResponse } from "next/server";

export async function POST(req: Request) {
  console.log("я зашел сюды");
  try {
    // Моки для проверки
    if (process.env.FLAG === "development") {
      const mockUrls: UploadImagesResponse = {
        message: "success",
        results: [
          {
            id: 0,
            image: "https://i.imgur.com/YzFSzED.jpeg",
            created_at: "",
          },
          {
            id: 1,
            image: "https://i.imgur.com/3giN25k.jpeg",
            created_at: "",
          },
        ],
      };

      return NextResponse.json<UploadImagesResponse>(mockUrls);
    }

    // Прод
    const formData = await req.formData();
    const files = formData.getAll("images") as File[];

    const accessToken = (await cookies()).get("access_token")?.value;
    console.log(formData);
    console.log(files);
    const djangoResponse = await fetch(
      `${process.env.DJANGO_API}/api/upload/`,
      {
        method: "POST",
        headers: {
          Authorization: `Bearer ${accessToken}`,
        },
        body: formData,
      }
    );

    if (!djangoResponse.ok) {
      throw new Error("Ошибка загрузки изображений");
    }

    const data = (await djangoResponse.json()) as UploadImagesResponse;
    console.log(data);
    const pre_res = NextResponse.json(data);

    return pre_res;
  } catch (error) {
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "Unknown error" },
      { status: 500 }
    );
  }
}
