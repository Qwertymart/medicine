import styles from "./moreinfo.module.css";
import LegendAccordion from "../LegendAccordion/LegendAccordion";
import { motion } from "framer-motion";
import Image from "next/image";

interface ModalInfoProps {
  pic: Result;
  onClose: () => void;
}

type ItemType = {
  title: string;
  description: string;
  color: string;
};

const legendItems: ItemType[] = [
  {
    title: "Гарь",
    description: "Территория, пострадавшая от лесного пожара.",
    color: "#FF0000",
  },
  {
    title: "Пашни",
    description: "Земли, используемые для сельскохозяйственных работ.",
    color: "#8b4513",
  },
  {
    title: "Леса",
    description: "Натуральный лесной массив, охраняемый от вырубки.",
    color: "#228b22",
  },
];

export default function MoreInfo({ pic, onClose }: ModalInfoProps) {
    console.log(pic);
  return (
    <motion.div
      className={styles.modalOverlay}
      onClick={onClose}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.2 }}
    >
      <motion.div
        className={styles.modalContent}
        onClick={(e) => e.stopPropagation()}
        initial={{ scale: 0.8, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        exit={{ scale: 0.8, opacity: 0 }}
        transition={{ type: "spring", stiffness: 300, damping: 20 }}
      >
        <motion.button
          className={styles.closeButton}
          onClick={onClose}
          whileHover={{ scale: 1.1 }}
          whileTap={{ scale: 0.9 }}
        >
          &times;
        </motion.button>
        <h3>Подробная информация</h3>
        <div className={styles.infoGrid}>
          <Image
            src={pic.image}
            alt="Файл"
            width={347}
            height={347}
            // fill
            className={styles.image}
          />
          <LegendAccordion legendItems={legendItems} />
          <p>Бла бла бла немножко текста тут вроде еще что-то бла бла бла</p>
        </div>
      </motion.div>
    </motion.div>
  );
}
