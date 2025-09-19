"use client";
import { useAuth } from "@/hooks/useAuth";
import styles from "./page.module.css";
import dynamic from "next/dynamic";

const Profile = dynamic(() => import("@/components/profile/profile"));
const UserPhotos = dynamic(() => import("@/components/userPhotos/userPhotos"));

export default function DahboardPage() {
  const { curUser, isLoading } = useAuth();

  console.log("User data:", curUser);

  // TODO: тут можно красоты навалить
  if (isLoading) {
    console.log("Загрузка");
    return <div>Загрузка...</div>;
  }

  if (curUser == null) {
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
          {curUser.photos?.length === 0 ? (
            <UserPhotos pics={curUser.photos} />
          ) : (
            <div style={{color: "black"}}>
              <p>пока тут пусто, но вы можете перейти на форму</p>
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
