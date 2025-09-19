import styles from './footer.module.css';
import Link from 'next/link';

export default function Footer() {
    return (
        <footer>
            <div className={styles.container}>
                <div className={styles.footerContent}>
                    <div className={styles.footerInfo}>
                        <h2 className={styles.footer__header}>О нас</h2>
                        <span className={styles.footer__text}>
                            Мы используем передовые алгоритмы машинного обучения и анализ больших данных для обработки
                            спутниковых снимков Приморского края. Наш сервис обеспечивает высокую точность распознавания
                            лесных участков, вырубок, гарей и пашни, предоставляя актуальную информацию для мониторинга
                            лесных ресурсов.
                        </span>
                    </div>
                    <div className={styles.footer__contact}>
                        <h2 className={styles.footer__header}>Контакты</h2>
                        <div className={styles.footerContactsWrapper}>
                            <Link className={styles.footerLinkfooterLinkBold} href="tel:88000000000">
                                <img className={styles.footerLinkImage} src="images/phone3.png" width="12" height="12"
                                     alt="Иконка телефона"/>
                                <span className={styles.footerLinkText}>8 (800) 000-00-00</span>
                            </Link>
                            <Link className={styles.footerLinkfooterLinkBold} href="mailto:email@email.com">
                                <img className={styles.footerLinkImage} src="images/mail3.png" width="12" height="12"
                                     alt="Иконка почты"/>
                                <span className={styles.footerLinkText}>email@email.com</span>
                            </Link>
                            <Link className={styles.footerLinkfooterLinkBold} href="/">
                                <img className={styles.footerLinkImage} src="images/point3.png" width="12" height="12"
                                     alt="Иконка навигации"/>
                                <span className={styles.footerLinkText}>г. Москва, ул. Петровско-Разумовская, 145, оф. 34</span>
                            </Link>
                        </div>
                    </div>
                </div>
            </div>
        </footer>
    );
}