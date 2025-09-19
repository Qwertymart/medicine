"use client";
import React, { useState } from "react";
import { motion } from "framer-motion";
import styles from "./styles.module.css";

type ItemType = {
  title: string;
  description: string;
  color: string;
};

const LegendItem = ({ title, description, color }: ItemType) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className={styles.legendContainer}>
      <button
        className={styles.legendHeader}
        onClick={() => setIsOpen((prev) => !prev)}
      >
        <svg
          className={styles.markerIcon}
          viewBox="0 0 19 19"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <circle cx="9.5" cy="9.5" r="9.5" fill={color} />
        </svg>
        <span className={styles.legendTitle}>{title}</span>
        <motion.span
          className={styles.arrow}
          animate={{ rotate: isOpen ? 180 : 0 }}
          transition={{ duration: 0.3 }}
        >
          ▼
        </motion.span>
      </button>

      <motion.div
        className={styles.legendContent}
        initial={{ height: 0, opacity: 0 }}
        animate={{ height: isOpen ? "auto" : 0, opacity: isOpen ? 1 : 0 }}
        transition={{ duration: 0.3 }}
      >
        <p className={styles.legendDescription}>{description}</p>
      </motion.div>
    </div>
  );
};

const LegendAccordion: React.FC<{ legendItems: ItemType[] }> = ({
  legendItems,
}: {
  legendItems: ItemType[];
}) => {
  return (
    <div>
      {legendItems.map((item, index) => (
        <LegendItem key={index} {...item} />
      ))}
    </div>
  );
};

export default LegendAccordion;
