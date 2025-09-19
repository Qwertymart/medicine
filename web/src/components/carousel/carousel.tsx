"use client";
import { AnimatePresence, motion } from "framer-motion";
import { useState } from "react";
import Image from "next/image";
import styles from "./styles.module.css"

export default function Carousel({ previewUrls }: { previewUrls: string[] }) {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [direction, setDirection] = useState<"left" | "right">("right");

  const handleNext = () => {
    setDirection("right");
    setCurrentIndex((prev) => (prev === previewUrls.length - 1 ? 0 : prev + 1));
  };

  const handlePrev = () => {
    setDirection("left");
    setCurrentIndex((prev) => (prev === 0 ? previewUrls.length - 1 : prev - 1));
  };

  const variants = {
    enter: (direction: string) => ({
      x: direction === "right" ? 100 : -100,
      opacity: 0,
    }),
    center: {
      x: 0,
      opacity: 1,
    },
    exit: (direction: string) => ({
      x: direction === "right" ? -100 : 100,
      opacity: 0,
    }),
  };

  return (
    <div className={styles.carousel}>
      <button
        type="button"
        className={styles.arrowButton}
        onClick={handlePrev}
        disabled={currentIndex === 0}
      >
        <Image
          src="/icons/right.svg"
          alt="Previous"
          width={24}
          height={24}
        />
      </button>

      <div className={styles.imageContainer}>
        <AnimatePresence mode="popLayout" initial={false} custom={direction}>
          <motion.div
            key={currentIndex}
            custom={direction}
            variants={variants}
            initial="enter"
            animate="center"
            exit="exit"
            transition={{
              type: "spring",
              stiffness: 300,
              damping: 30,
            }}
            className={styles.motionWrapper}
          >
            <Image
              src={previewUrls[currentIndex]}
              alt={`Slide ${currentIndex + 1}`}
              fill
              // width={600}
              // height={400}
              className={styles.carouselImage}
              style={{cursor: "default"}}
              // sizes="(max-width: 768px) 100vw, 80vw"
              // priority
            />
          </motion.div>
        </AnimatePresence>
      </div>

      <button
        type="button"
        className={styles.arrowButton}
        onClick={handleNext}
        disabled={currentIndex === previewUrls.length - 1}
      >
        <Image src="/icons/left.svg" alt="Next" width={24} height={24} />
      </button>
    </div>
  );
}
