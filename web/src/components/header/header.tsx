import styles from './header.module.css';
import Link from 'next/link';

export default function Header() {

    return (
        <header className={styles.header}>
            <div className={styles.container}>
                <div className={styles.content}>
                    <Link className={styles.link} href="/">
                        <img className={styles.logo} src="images/logo3.png" alt="Логотип"/>
                    </Link>

                    <div className={styles.middle}>
                        <form className={styles.search} action="#" method="GET">
                            <input className={styles.searchField} type="search" aria-label="Поле поиска"
                                   name="search-field"
                                   placeholder="поиск"/>
                            {/* <svg className={styles.searchIcon} width="12" height="12" aria-hidden="true">
                        <use xlink:href="images/sprite.svg#icon-search"></use>
                    </svg> */}
                        </form>
                    </div>

                    <ul className={styles.nav}>
                        <li className={styles.navItem}>
                            <Link className={styles.navLink} href="#instructionSection">Инструкция</Link>
                        </li>
                        <li className={styles.navItem}>
                            <Link className={styles.navLink} href="/upload">Форма</Link>
                        </li>
                        <li className={styles.navItem}>
                            <Link className={styles.navLink} href="/register">Регистрация</Link>
                        </li>
                        <li className={styles.navItem}>
                            <Link className={styles.navLink} href="/dashboard">Личный кабинет</Link>
                        </li>
                    </ul>
                </div>
            </div>
        </header>
    );
}