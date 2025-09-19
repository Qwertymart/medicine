import Link from "next/link";
import styles from "./notfound.module.css";


export default function NotFound(){
    return (
  <div className={styles.container}>
    <h1 className={styles.error404}>404</h1>
    <p className={styles.ups}>Упс! Страница не найдена.</p>
    <p className={styles.ups}>Вернитесь на <Link className={styles.cancel} href="/">главную страницу</Link>.</p>
  </div>
    )
}