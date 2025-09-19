"use client";
import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";

export const useAuth = () => {
  const router = useRouter();
  const pathname = usePathname();
  const [curUser, setCurUser] = useState({
    firstName: "",
    lastName: "",
    username: "",
    email: "",
    avatarSrc: "",
    photos: [],
  });
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const PUBLIC_PATHS = ["/", "/login", "/register"];
    if (PUBLIC_PATHS.includes(pathname)) {
      setIsLoading(false);
      console.log("1");
      return;
    }
    let isMounted = true;
    const controller = new AbortController();

    const fetchAuthData = async () => {
      try {
        const authCheck = await fetch("/api/auth/check", {
          credentials: "include",
          signal: controller.signal,
          headers: { "Cache-Control": "no-cache" },
        });

        if (!authCheck.ok) {
          router.push("/login");
          // throw new Error("Unauthorized");
        }

        const userResponse = await fetch("/api/auth/get_user", {
          credentials: "include",
          signal: controller.signal,
        });

        if (!userResponse.ok) throw new Error("Failed to fetch user");

        const userData = await userResponse.json();

        if (isMounted) {
          setCurUser({
            firstName: userData.firstName || "",
            lastName: userData.lastName || "",
            username: userData.username || "",
            email: userData.email || "",
            avatarSrc: userData.avatarSrc || "",
            photos: userData.photos || [],
          });
          setIsLoading(false);
        }
      } catch (error) {
        if (isMounted) {
          console.error("Auth error:", error);
          if (!PUBLIC_PATHS.includes(pathname)) {
            console.log("Dangerous place!!!");
            router.replace("/login");
          }
          setIsLoading(false);
        }
      }
    };

    fetchAuthData();

    return () => {
      isMounted = false;
      controller.abort();
    };
  }, [router, pathname]);

  return { curUser, isLoading };
};
