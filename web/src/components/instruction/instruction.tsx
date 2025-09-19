import Link from 'next/link';
import styles from './instruction.module.css';

export default function Instruction() {
    return (
        <section className={styles.instructionWrapper} id="instructionSection">
            {/* Левый блок */}
            <div className={styles.instructionWrapperLeft}>
                <div className={styles.instructionLinkPosition}>
                    <Link className={styles.instructionLink} href="instructionSection">
                        Как пользоваться?
                    </Link>
                </div>
                <div className={styles.instructionLinkPosition}>
                    <h3 className={styles.instructionHeader}>Инструкция по использованию</h3>
                </div>
                <div className={styles.instructionLinkPosition}>
                    <span className={styles.instructionText}>
                        Мы подробно рассмотрим, как использовать сервис{"\n"}
                        <b>ForestWatch Приморье</b> для распознавания участков
                    </span>
                </div>
            </div>

            {/* Центральный блок */}
            <div className={styles.instructionWrapperCentral}>
                <ul className={styles.instructionWrapperSteps}>
                    <li className={styles.instructionWrapperStepsItem}>
                        <h3 className={styles.instructionWrapperStepsNumber}>01</h3>
                        <span className={styles.instructionText}>
                            Откройте вкладку Форма для загрузки
                        </span>
                    </li>
                    <li className={styles.instructionWrapperStepsItem}>
                        <h3 className={styles.instructionWrapperStepsNumber}>02</h3>
                        <span className={styles.instructionText}>
                            Добавьте один или несколько файлов формата: <b>JPG, PNG</b>
                        </span>
                    </li>
                    <li className={styles.instructionWrapperStepsItem}>
                        <h3 className={styles.instructionWrapperStepsNumber}>03</h3>
                        <span className={styles.instructionText}>
                            Нажмите кнопку отправить и наблюдайте результат в форме Ваш результат
                        </span>
                    </li>
                    <li className={styles.instructionWrapperStepsItem}>
                        <h3 className={styles.instructionWrapperStepsNumber}>04</h3>
                        <div>
                            <span className={styles.instructionText}>
                                Цвета выделенных участков:
                            </span>
                            <ul className={styles.instructionWrapperColors}>
                                {/*<li className={`${styles.instructionWrapperColorsItem} ${styles.instructionWrapperColorsItemRed}`}>*/}
                                {/*    <span className={styles.instructionWrapperColorsName}>Гарь</span>*/}
                                {/*</li>*/}
                                <li className={`${styles.instructionWrapperColorsItem} ${styles.itemForest}`}>
                                    <span className={styles.instructionWrapperColorsName}>Forest - Лес</span>
                                </li>
                                <li className={`${styles.instructionWrapperColorsItem} ${styles.itemFelling}`}>
                                    <span className={styles.instructionWrapperColorsName}>Felling - Вырубка</span>
                                </li>
                                <li className={`${styles.instructionWrapperColorsItem} ${styles.itemPlow}`}>
                                    <span className={styles.instructionWrapperColorsName}>Plow - Сельскохозяйственное угодье</span>
                                </li>
                            </ul>
                        </div>
                    </li>
                </ul>
            </div>

            {/* Правый блок */}
            <div className={styles.instructionWrapperRight}>
                <img
                    className={styles.instructionWrapperImage}
                    src="images/map.jpg"
                    width="390"
                    height="275"
                    alt="Карта"
                />
            </div>
        </section>
    );
}