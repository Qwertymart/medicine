import styles from './hero.module.css';
import Link from 'next/link';

export default function Hero(){
    return (
        <section className={styles.hero}>
            <div className={styles.container}>
                <div className={styles.heroContent}>
                    {/* Не нужная здесь была хрень*/}
                    {/*<h1 className={styles.heroHeadervisuallyHidden}>ForestWatch Приморье</h1>*/}

                    <div className={styles.heroWrapper}>
                        <img className={styles.heroImage} src="images/logo5.png" width="345" height="125" alt="Логотип ForestWatch"/>
                        <span className={styles.heroText}>Посмотри инструкцию и&nbsp;начни прямо сейчас</span>
                        <Link className={styles.heroLink} href="/dashboard">Начать</Link>

                        <ul className={styles.heroPages}>
                            <li className={styles.heroPagesItemheroPpagesItemActive}></li>
                            <li className={styles.heroPagesItem}></li>
                            <li className={styles.heroPagesItem}></li>
                        </ul>
                    </div>
                    <img className={styles.heroLogo} src="images/bird.png" width="340" height="244" alt="Птичка"/>
                </div>
            </div>
        </section>
); 
}