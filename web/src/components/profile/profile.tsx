"use client";

import Button from "@/components/button/button";
import styles from "./profile.module.css";
import Image from "next/image";
import { useRouter } from "next/navigation";

export default function Profile({ user }: { user: UserInfo }) {
  const router = useRouter();

  async function handleExit(): Promise<void> {
    try {
      const response = await fetch("/api/auth", {
        method: "POST",
        body: JSON.stringify({
          action: "logout",
        }),
      });
      if (response.ok) {
        router.replace("/login");
      } else {
        console.error("Ошибка при выходе");
      }
    } catch (error) {
      console.error("Ошибка при выходе", error);
    }
  }

  return (
    <div className={styles.profile}>
      <div className={styles.profileHeader}>
        <Image
          className={styles.avatar}
          src={user.avatarSrc}
          width={174}
          height={174}
          alt="avatar"
        />

        <h3 className={styles.userName}>
          {user.lastName} {user.firstName}
        </h3>
        <div>
          {" "}
          <Button
            variant="white"
            size="large"
            onClick={handleExit}
            // className={styles.exitBtn}
          >
            Выйти
          </Button>
        </div>
      </div>
      <div className={styles.profileInfo}>
        <p>
          <strong>Логин:</strong> {user.username}
        </p>
        <p>
          <strong>Почта:</strong> {user.email}
        </p>
      </div>
    </div>
  );
}
