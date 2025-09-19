import styles from "./page.module.css";
import dynamic from "next/dynamic";

const UploadCarousel = dynamic(
  () => import("@/components/card/UploadCard"));

export default function UploadPage() {
  return (
    <div className={styles.main}>
      <UploadCarousel />
    </div>
  );
}