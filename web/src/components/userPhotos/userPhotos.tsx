"use client";

import styles from "./userPhotos.module.css";
import dynamic from "next/dynamic";

const UserPic = dynamic(() => import("@/components/userpic/userpic"));


export default function UserPhotos({ pics }: { pics: Result[] }) {
  console.log(pics[0].image);
  return (
    <section className={styles.container}>
      <h2 className={styles.title}>Готовые файлы</h2>
      <div className={styles.grid}>
        {pics.map((elem, index) => (
          <UserPic key={index} pic={elem} />
        ))}
      </div>
    </section>
  );
}
