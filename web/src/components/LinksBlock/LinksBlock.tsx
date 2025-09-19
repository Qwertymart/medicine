import classNames from "classnames";
import styles from "./style.module.css";
import Link from "next/link";
import { usePathname } from "next/navigation";

const LinksBlock = () => {
  const curPath = usePathname();
  return (
    <div className={styles.linksBlock}>
      <Link
        href="/login"
        className={classNames(
          styles.link,
          curPath === "/login" && styles.activeLink
        )}
      >
        Войти
      </Link>
      <Link
        href="/register"
        className={classNames(
          styles.link,
          curPath === "/register" && styles.activeLink
        )}
      >
        Регистрация
      </Link>
    </div>
  );
};

export default LinksBlock;
