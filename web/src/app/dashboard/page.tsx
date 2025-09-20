"use client";
import { useAuth } from "@/hooks/useAuth";
import styles from "./page.module.css";
import dynamic from "next/dynamic";
import { useEffect } from "react";

const Profile = dynamic(() => import("@/components/profile/profile"));
const UserPhotos = dynamic(() => import("@/components/userPhotos/userPhotos"));

export default function DahboardPage() {
  const { curUser, isLoading } = useAuth();

  useEffect(() => {
    console.log("User data:", curUser);
  }, [curUser]);

  // TODO: тут можно красоты навалить
  if (isLoading) {
    console.log("Загрузка");
    return <div>Загрузка...</div>;
  }

  if (curUser) {
    console.log();
  }

  if (!curUser?.email) {
    console.log(
      "Ошибка авторизации! Пользователь не найден или не авторизован"
    );
  }

  return (
    <div className={styles.container}>
      <div className={styles.page}>
        <div className={styles.resultsZone}>
          {curUser.photos?.length > 0 ? (
            <UserPhotos pics={curUser.photos} />
          ) : (
            <div style={{ color: "black" }}>
              <p>пока тут пусто, но вы можете перейти на форму и загрузить</p>
            </div>
          )}
        </div>
        <div className={styles.profileZone}>
          <Profile user={curUser} />
        </div>
      </div>
    </div>
  );
}
